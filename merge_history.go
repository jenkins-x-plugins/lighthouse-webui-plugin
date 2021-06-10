package webui

import (
	"encoding/json"
	"regexp"
	"time"

	"github.com/Jeffail/gabs/v2"
)

// from lighthouse/pkg/keeper/history.Record
type MergeRecord struct {
	Owner      string
	Repository string
	Branch     string

	Time    time.Time
	Action  string
	BaseSHA string

	PRs []PullRequest

	// this is the original keeper object
	KeeperRecord interface{}
}

func MergeRecordsFromLighthouseRecords(lhRecords *gabs.Container) []MergeRecord {
	if lhRecords == nil {
		return nil
	}

	var records []MergeRecord
	for name, children := range lhRecords.ChildrenMap() {
		org, repo, branch := parseHistoryPoolName(name)
		for _, child := range children.Children() {
			child.Set(org, "Org")
			child.Set(repo, "Repo")
			child.Set(branch, "Branch")
			record := MergeRecordFromLighthouseRecord(child)
			records = append(records, record)
		}
	}
	return records
}

func MergeRecordFromLighthouseRecord(lhRecord *gabs.Container) MergeRecord {
	record := MergeRecord{
		Owner:        lhRecord.Search("Org").Data().(string),
		Repository:   lhRecord.Search("Repo").Data().(string),
		Branch:       lhRecord.Search("Branch").Data().(string),
		Action:       lhRecord.Search("action").Data().(string),
		KeeperRecord: lhRecord.Data(),
	}

	timeStr := lhRecord.Search("time").Data().(string)
	if t, err := time.Parse(time.RFC3339Nano, timeStr); err == nil {
		record.Time = t
	}

	for _, target := range lhRecord.Search("target").Children() {
		pr := PullRequest{
			Author: target.Search("author").Data().(string),
			Title:  target.Search("title").Data().(string),
		}
		number := target.Search("number").Data().(json.Number)
		if n, err := number.Int64(); err == nil {
			pr.Number = int(n)
		}
		record.PRs = append(record.PRs, pr)
	}

	return record
}

// org/repo:branch
var keeperHistoryPoolRegex = regexp.MustCompile(`(?P<org>[^/]+)/(?P<repo>[^:]+):(?P<branch>.+)`)

func parseHistoryPoolName(name string) (org, repo, branch string) {
	allMatches := keeperHistoryPoolRegex.FindAllStringSubmatch(name, 1)
	if len(allMatches) == 0 {
		return
	}

	matches := allMatches[0]
	if len(matches) < 4 {
		return
	}

	org = matches[1]
	repo = matches[2]
	branch = matches[3]
	return
}
