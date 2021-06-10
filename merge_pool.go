package webui

import (
	"encoding/json"
	"time"

	"github.com/Jeffail/gabs/v2"
)

// from lighthouse/pkg/keeper.Pool
type MergePool struct {
	Owner      string
	Repository string
	Branch     string

	// PRs with passing tests, pending tests, and missing or failed tests.
	// Note that these results are rolled up. If all tests for a PR are passing
	// except for one pending, it will be in PendingPRs.
	SuccessPRs []PullRequest
	PendingPRs []PullRequest
	MissingPRs []PullRequest

	// Empty if there is no pending batch.
	BatchPending []PullRequest

	// this is the most recent UpdatedAt field from the different PRs
	UpdatedAt time.Time

	Action   string
	Target   []PullRequest
	Blockers []BlockerIssue
	Error    string

	// this is the original keeper object
	KeeperPool interface{}
}

type PullRequest struct {
	Number    int
	Author    string
	Mergeable string
	Title     string
	UpdatedAt time.Time
}

type BlockerIssue struct {
	Number int
	Title  string
	URL    string
}

func MergePoolsFromLighthousePools(lhPools *gabs.Container) []MergePool {
	if lhPools == nil {
		return nil
	}

	var pools []MergePool
	for _, child := range lhPools.Children() {
		pool := MergePoolFromLighthousePool(child)
		pools = append(pools, pool)
	}
	return pools
}

func MergePoolFromLighthousePool(lhPool *gabs.Container) MergePool {
	pool := MergePool{
		Owner:      lhPool.Search("Org").Data().(string),
		Repository: lhPool.Search("Repo").Data().(string),
		Branch:     lhPool.Search("Branch").Data().(string),
		Action:     lhPool.Search("Action").Data().(string),
		Error:      lhPool.Search("Error").Data().(string),
		KeeperPool: lhPool.Data(),
	}

	for _, child := range lhPool.Search("SuccessPRs").Children() {
		pr := PullRequestFromLighthousePullRequest(child)
		if pr.UpdatedAt.After(pool.UpdatedAt) {
			pool.UpdatedAt = pr.UpdatedAt
		}
		pool.SuccessPRs = append(pool.SuccessPRs, pr)
	}
	for _, child := range lhPool.Search("PendingPRs").Children() {
		pr := PullRequestFromLighthousePullRequest(child)
		if pr.UpdatedAt.After(pool.UpdatedAt) {
			pool.UpdatedAt = pr.UpdatedAt
		}
		pool.PendingPRs = append(pool.PendingPRs, pr)
	}
	for _, child := range lhPool.Search("MissingPRs").Children() {
		pr := PullRequestFromLighthousePullRequest(child)
		if pr.UpdatedAt.After(pool.UpdatedAt) {
			pool.UpdatedAt = pr.UpdatedAt
		}
		pool.MissingPRs = append(pool.MissingPRs, pr)
	}
	for _, child := range lhPool.Search("BatchPending").Children() {
		pr := PullRequestFromLighthousePullRequest(child)
		if pr.UpdatedAt.After(pool.UpdatedAt) {
			pool.UpdatedAt = pr.UpdatedAt
		}
		pool.BatchPending = append(pool.BatchPending, pr)
	}
	for _, child := range lhPool.Search("Target").Children() {
		pr := PullRequestFromLighthousePullRequest(child)
		if pr.UpdatedAt.After(pool.UpdatedAt) {
			pool.UpdatedAt = pr.UpdatedAt
		}
		pool.Target = append(pool.Target, pr)
	}

	return pool
}

func PullRequestFromLighthousePullRequest(lhPR *gabs.Container) PullRequest {
	pr := PullRequest{
		Author:    lhPR.Path("Author.Login").Data().(string),
		Mergeable: lhPR.Search("Mergeable").Data().(string),
		Title:     lhPR.Search("Title").Data().(string),
	}

	number := lhPR.Search("Number").Data().(json.Number)
	if n, err := number.Int64(); err == nil {
		pr.Number = int(n)
	}

	timeStr := lhPR.Search("UpdatedAt").Data().(string)
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		pr.UpdatedAt = t
	}

	return pr
}
