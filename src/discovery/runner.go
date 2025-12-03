package discovery

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	activeSuites = make(map[string]*CheckSuite)
	suitesMu     sync.RWMutex
)

func NewCheckSuite(url string) *CheckSuite {
	return &CheckSuite{
		Id:        uuid.New().String(),
		Status:    CheckStatusPending,
		StartTime: time.Now(),
		cancel:    make(chan struct{}),
		CheckURL:  url,
	}
}

func GetCheckSuite(id string) (*CheckSuite, bool) {
	suitesMu.RLock()
	defer suitesMu.RUnlock()
	suite, ok := activeSuites[id]
	return suite, ok
}

func CancelCheckSuite(id string) error {
	suitesMu.Lock()
	defer suitesMu.Unlock()

	suite, ok := activeSuites[id]
	if !ok {
		return nil
	}

	if suite.Status == CheckStatusRunning {
		close(suite.cancel)
		suite.Status = CheckStatusCanceled
	}

	return nil
}

func (ts *CheckSuite) GetSnapshot() *CheckSuite {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	return &CheckSuite{
		Id:                     ts.Id,
		Status:                 ts.Status,
		StartTime:              ts.StartTime,
		EndTime:                ts.EndTime,
		TotalChecks:            ts.TotalChecks,
		CompletedChecks:        ts.CompletedChecks,
		SuccessfulChecks:       ts.SuccessfulChecks,
		FailedChecks:           ts.FailedChecks,
		CheckURL:               ts.CheckURL,
		DomainDiscoveryResults: ts.DomainDiscoveryResults,
		CurrentPhase:           ts.CurrentPhase,
	}
}
