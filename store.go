package webui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/search"
	"github.com/blevesearch/bleve/search/query"
	"github.com/sirupsen/logrus"
)

const (
	// eventsIndexMappingVersion is the version of the events index mapping
	// if you change something in the mapping, increment this version
	// this is used to ensure the index will be recreated if we change the mapping
	eventsIndexMappingVersion = 1
)

type Store struct {
	config            StoreConfig
	gcStopChan        chan struct{}
	events            bleve.Index
	jobs              bleve.Index
	mergeStatus       []MergePool
	mergeStatusMutex  sync.RWMutex
	mergeHistory      []MergeRecord
	mergeHistoryMutex sync.RWMutex
}

type StoreConfig struct {
	DataPath     string
	MaxEvents    int
	EventsMaxAge time.Duration
}

func NewStore(cfg StoreConfig, logger *logrus.Logger) (*Store, error) {
	var (
		store = new(Store)
		err   error
	)

	keywordFieldMapping := bleve.NewTextFieldMapping()
	keywordFieldMapping.Analyzer = keyword.Name

	jobsMapping := bleve.NewIndexMapping()
	jobsMapping.DefaultAnalyzer = keyword.Name
	jobsMapping.DefaultMapping.AddFieldMappingsAt("Start", bleve.NewDateTimeFieldMapping())
	jobsMapping.DefaultMapping.AddFieldMappingsAt("End", bleve.NewDateTimeFieldMapping())

	eventsMapping := bleve.NewIndexMapping()
	eventsMapping.DefaultAnalyzer = keyword.Name
	eventsMapping.DefaultMapping.AddFieldMappingsAt("Time", bleve.NewDateTimeFieldMapping())

	store.jobs, err = bleve.NewMemOnly(jobsMapping)
	if err != nil {
		return nil, fmt.Errorf("failed to created a Bleve in-memory Index: %w", err)
	}

	if cfg.DataPath == "" {
		store.events, err = bleve.NewMemOnly(eventsMapping)
		if err != nil {
			return nil, fmt.Errorf("failed to created a Bleve in-memory Index: %w", err)
		}
	} else {
		eventsDataPath := filepath.Join(cfg.DataPath, fmt.Sprintf("events-v%d", eventsIndexMappingVersion))
		store.events, err = bleve.Open(eventsDataPath)
		if errors.Is(err, bleve.ErrorIndexPathDoesNotExist) {
			store.events, err = bleve.New(eventsDataPath, eventsMapping)
		} else if err != nil {
			if logger != nil {
				logger.WithError(err).WithField("index-path", eventsDataPath).Warning("failed to open existing Bleve index - a new (empty) index will be created...")
			}
			store.events, err = bleve.New(eventsDataPath, eventsMapping)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to created a Bleve Index at %s: %w", eventsDataPath, err)
		}
	}

	store.config = cfg
	store.gcStopChan = make(chan struct{})

	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				store.CollectGarbage()
			case <-store.gcStopChan:
				logger.Info("Store GarbageCollector exiting...")
				return
			}
		}
	}()

	return store, nil
}

func (s *Store) Close() error {
	close(s.gcStopChan)
	return s.events.Close()
}

func (s *Store) SetMergeStatus(pools []MergePool) {
	s.mergeStatusMutex.Lock()
	defer s.mergeStatusMutex.Unlock()
	s.mergeStatus = make([]MergePool, len(pools))
	copy(s.mergeStatus, pools)
}

func (s *Store) QueryMergeStatus(q MergeStatusQuery) []MergePool {
	s.mergeStatusMutex.RLock()
	defer s.mergeStatusMutex.RUnlock()

	var pools []MergePool
	for _, pool := range s.mergeStatus {
		if q.Owner != "" && q.Owner != pool.Owner {
			continue
		}
		if q.Repository != "" && q.Repository != pool.Repository {
			continue
		}
		if q.Branch != "" && q.Branch != pool.Branch {
			continue
		}
		pools = append(pools, pool)
	}
	return pools
}

func (s *Store) SetMergeHistory(records []MergeRecord) {
	s.mergeHistoryMutex.Lock()
	defer s.mergeHistoryMutex.Unlock()
	s.mergeHistory = make([]MergeRecord, len(records))
	copy(s.mergeHistory, records)
}

func (s *Store) QueryMergeHistory(q MergeHistoryQuery) []MergeRecord {
	s.mergeHistoryMutex.RLock()
	defer s.mergeHistoryMutex.RUnlock()

	var records []MergeRecord
	for _, record := range s.mergeHistory {
		if q.Owner != "" && q.Owner != record.Owner {
			continue
		}
		if q.Repository != "" && q.Repository != record.Repository {
			continue
		}
		if q.Branch != "" && q.Branch != record.Branch {
			continue
		}
		records = append(records, record)
	}
	return records
}

func (s *Store) AddJob(j Job) error {
	return s.jobs.Index(j.Name, j)
}

func (s *Store) DeleteJob(name string) error {
	return s.jobs.Delete(name)
}

func (s *Store) AddEvent(e Event) error {
	return s.events.Index(e.GUID, e)
}

func (s *Store) QueryJobs(q JobsQuery) (*Jobs, error) {
	request := bleve.NewSearchRequest(q.ToBleveQuery())
	request.SortBy([]string{"-Start"})
	request.Size = 10000
	request.Fields = []string{"*"}
	request.AddFacet("State", bleve.NewFacetRequest("State", 4))
	request.AddFacet("Repository", bleve.NewFacetRequest("Repository", 3))
	request.AddFacet("Type", bleve.NewFacetRequest("Type", 3))
	request.AddFacet("Author", bleve.NewFacetRequest("Author", 3))
	result, err := s.jobs.Search(request)
	if err != nil {
		return nil, fmt.Errorf("failed to search for %v: %w", q, err)
	}

	jobs := bleveResultToJobs(result)
	return &jobs, nil
}

func (s *Store) QueryEvents(q EventsQuery) (*Events, error) {
	request := bleve.NewSearchRequest(q.ToBleveQuery())
	request.SortBy([]string{"-Time"})
	request.Size = 10000
	request.Fields = []string{"*"}
	request.AddFacet("Kind", bleve.NewFacetRequest("Kind", 4))
	request.AddFacet("Repository", bleve.NewFacetRequest("Repository", 3))
	request.AddFacet("Sender", bleve.NewFacetRequest("Sender", 3))
	result, err := s.events.Search(request)
	if err != nil {
		return nil, fmt.Errorf("failed to search for %v: %w", q, err)
	}

	events := bleveResultToEvents(result)
	return &events, nil
}

func (s *Store) CollectGarbage() error {
	var deleteMatchingEvents = func(req *bleve.SearchRequest) error {
		result, err := s.events.Search(req)
		if err != nil {
			return err
		}
		for _, doc := range result.Hits {
			if err = s.events.Delete(doc.ID); err != nil {
				return err
			}
		}
		return nil
	}
	if s.config.MaxEvents > 0 {
		request := bleve.NewSearchRequest(bleve.NewMatchAllQuery())
		request.SortBy([]string{"-Time"})
		request.Size = 1000
		request.From = s.config.MaxEvents
		if err := deleteMatchingEvents(request); err != nil {
			return err
		}
	}
	if s.config.EventsMaxAge > 0 {
		request := bleve.NewSearchRequest(bleve.NewDateRangeQuery(time.Time{}, time.Now().Add(-s.config.EventsMaxAge)))
		request.Size = 1000
		if err := deleteMatchingEvents(request); err != nil {
			return err
		}
	}
	return nil
}

type JobsQuery struct {
	EventGUID  string
	Owner      string
	Repository string
	Branch     string
	Query      string
}

func (q JobsQuery) ToBleveQuery() query.Query {
	var queryString strings.Builder
	if len(q.Query) > 0 {
		queryString.WriteString("+")
		queryString.WriteString(q.Query)
	}
	if len(q.EventGUID) > 0 {
		if queryString.Len() > 0 {
			queryString.WriteString(" ")
		}
		queryString.WriteString("+EventGUID:")
		queryString.WriteString(q.EventGUID)
	}
	if len(q.Owner) > 0 {
		if queryString.Len() > 0 {
			queryString.WriteString(" ")
		}
		queryString.WriteString("+Owner:")
		queryString.WriteString(q.Owner)
	}
	if len(q.Repository) > 0 {
		if queryString.Len() > 0 {
			queryString.WriteString(" ")
		}
		queryString.WriteString("+Repository:")
		queryString.WriteString(q.Repository)
	}
	if len(q.Branch) > 0 {
		if queryString.Len() > 0 {
			queryString.WriteString(" ")
		}
		queryString.WriteString("+Branch:")
		queryString.WriteString(q.Branch)
	}
	if queryString.Len() == 0 {
		return bleve.NewMatchAllQuery()
	}
	return bleve.NewQueryStringQuery(queryString.String())
}

func bleveResultToJobs(result *bleve.SearchResult) Jobs {
	var jobs Jobs

	for _, doc := range result.Hits {
		job := bleveDocToJob(doc)
		jobs.Jobs = append(jobs.Jobs, job)
	}

	for _, facet := range result.Facets {
		counts := map[string]int{}
		for _, term := range facet.Terms {
			counts[term.Term] = term.Count
		}
		for _, numericRange := range facet.NumericRanges {
			counts[numericRange.Name] = numericRange.Count
		}
		counts["Other"] = facet.Other
		switch facet.Field {
		case "State":
			jobs.Counts.States = counts
		case "Repository":
			jobs.Counts.Repositories = counts
		case "Type":
			jobs.Counts.Types = counts
		case "Author":
			jobs.Counts.Authors = counts
		}
	}

	return jobs
}

func bleveDocToJob(doc *search.DocumentMatch) Job {
	var (
		startDate, endDate time.Time
	)
	if start, ok := doc.Fields["Start"].(string); ok {
		startDate, _ = time.Parse(time.RFC3339, start)
	}
	if end, ok := doc.Fields["End"].(string); ok {
		endDate, _ = time.Parse(time.RFC3339, end)
	}
	return Job{
		Name:        doc.Fields["Name"].(string),
		Type:        doc.Fields["Type"].(string),
		EventGUID:   doc.Fields["EventGUID"].(string),
		Owner:       doc.Fields["Owner"].(string),
		Repository:  doc.Fields["Repository"].(string),
		Branch:      doc.Fields["Branch"].(string),
		Build:       doc.Fields["Build"].(string),
		Context:     doc.Fields["Context"].(string),
		Author:      doc.Fields["Author"].(string),
		State:       doc.Fields["State"].(string),
		Description: doc.Fields["Description"].(string),
		ReportURL:   doc.Fields["ReportURL"].(string),
		TraceID:     doc.Fields["TraceID"].(string),
		Start:       startDate,
		End:         endDate,
		Duration:    time.Duration(doc.Fields["Duration"].(float64)),
	}
}

type EventsQuery struct {
	GUID       string
	Owner      string
	Repository string
	Branch     string
	Query      string
}

func (q EventsQuery) ToBleveQuery() query.Query {
	var queryString strings.Builder
	if len(q.Query) > 0 {
		queryString.WriteString("+")
		queryString.WriteString(q.Query)
	}
	if len(q.GUID) > 0 {
		if queryString.Len() > 0 {
			queryString.WriteString(" ")
		}
		queryString.WriteString("+GUID:")
		queryString.WriteString(q.GUID)
	}
	if len(q.Owner) > 0 {
		if queryString.Len() > 0 {
			queryString.WriteString(" ")
		}
		queryString.WriteString("+Owner:")
		queryString.WriteString(q.Owner)
	}
	if len(q.Repository) > 0 {
		if queryString.Len() > 0 {
			queryString.WriteString(" ")
		}
		queryString.WriteString("+Repository:")
		queryString.WriteString(q.Repository)
	}
	if len(q.Branch) > 0 {
		if queryString.Len() > 0 {
			queryString.WriteString(" ")
		}
		queryString.WriteString("+Branch:")
		queryString.WriteString(q.Branch)
	}
	if queryString.Len() == 0 {
		return bleve.NewMatchAllQuery()
	}
	return bleve.NewQueryStringQuery(queryString.String())
}

func bleveResultToEvents(result *bleve.SearchResult) Events {
	var events Events

	for _, doc := range result.Hits {
		event := bleveDocToEvent(doc)
		events.Events = append(events.Events, event)
	}

	for _, facet := range result.Facets {
		counts := map[string]int{}
		for _, term := range facet.Terms {
			counts[term.Term] = term.Count
		}
		for _, numericRange := range facet.NumericRanges {
			counts[numericRange.Name] = numericRange.Count
		}
		counts["Other"] = facet.Other
		switch facet.Field {
		case "Kind":
			events.Counts.Kinds = counts
		case "Repository":
			events.Counts.Repositories = counts
		case "Sender":
			events.Counts.Senders = counts
		}
	}

	return events
}

func bleveDocToEvent(doc *search.DocumentMatch) Event {
	var eventTime time.Time
	if evTime, ok := doc.Fields["Time"].(string); ok {
		eventTime, _ = time.Parse(time.RFC3339, evTime)
	}
	return Event{
		GUID:       doc.Fields["GUID"].(string),
		Owner:      doc.Fields["Owner"].(string),
		Repository: doc.Fields["Repository"].(string),
		Branch:     doc.Fields["Branch"].(string),
		Kind:       doc.Fields["Kind"].(string),
		Action:     doc.Fields["Action"].(string),
		Details:    doc.Fields["Details"].(string),
		URL:        doc.Fields["URL"].(string),
		Sender:     doc.Fields["Sender"].(string),
		Time:       eventTime,
	}
}

type MergeStatusQuery struct {
	Owner      string
	Repository string
	Branch     string
}

type MergeHistoryQuery struct {
	Owner      string
	Repository string
	Branch     string
}
