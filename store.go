package webui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/search"
	"github.com/blevesearch/bleve/search/query"
)

type Store struct {
	events            bleve.Index
	jobs              bleve.Index
	mergeStatus       []MergePool
	mergeStatusMutex  sync.RWMutex
	mergeHistory      []MergeRecord
	mergeHistoryMutex sync.RWMutex
}

func NewStore() (*Store, error) {
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

	store.events, err = bleve.NewMemOnly(eventsMapping)
	if err != nil {
		return nil, fmt.Errorf("failed to created a Bleve in-memory Index: %w", err)
	}

	return store, nil
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
	result, err := s.events.Search(request)
	if err != nil {
		return nil, fmt.Errorf("failed to search for %v: %w", q, err)
	}

	events := bleveResultToEvents(result)
	return &events, nil
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
		jobs = append(jobs, job)
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
		events = append(events, event)
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
		Details:    doc.Fields["Details"].(string),
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
