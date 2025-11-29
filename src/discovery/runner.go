package discovery

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/daniellavrushin/b4/log"
	"github.com/google/uuid"
)

var (
	activeSuites = make(map[string]*CheckSuite)
	suitesMu     sync.RWMutex
)

func NewCheckSuite(config CheckConfig) *CheckSuite {
	return &CheckSuite{
		Id:        uuid.New().String(),
		Status:    CheckStatusPending,
		StartTime: time.Now(),
		Results:   make([]CheckResult, 0),
		cancel:    make(chan struct{}),
		Config:    config,
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
		return fmt.Errorf("test suite not found")
	}

	if suite.Status == CheckStatusRunning {
		close(suite.cancel)
		suite.Status = CheckStatusCanceled
	}

	return nil
}

func (ts *CheckSuite) Run(domains []string) {
	suitesMu.Lock()
	activeSuites[ts.Id] = ts
	suitesMu.Unlock()

	defer func() {
		ts.EndTime = time.Now()
		ts.calculateSummary()

		// Keep in memory for 5 minutes
		time.AfterFunc(5*time.Minute, func() {
			suitesMu.Lock()
			delete(activeSuites, ts.Id)
			suitesMu.Unlock()
		})
	}()

	ts.mu.Lock()
	ts.Status = CheckStatusRunning
	ts.TotalChecks = len(domains)
	ts.mu.Unlock()

	log.Infof("Starting test suite %s with %d domains", ts.Id, len(domains))

	// Create worker pool
	semaphore := make(chan struct{}, ts.Config.MaxConcurrent)
	var wg sync.WaitGroup

	for _, sample := range domains {
		select {
		case <-ts.cancel:
			log.Infof("Check suite %s canceled", ts.Id)
			return
		default:
		}

		wg.Add(1)
		semaphore <- struct{}{} // Acquire

		go func(s string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release

			result := ts.testDomain(s)

			ts.mu.Lock()
			ts.Results = append(ts.Results, result)
			ts.CompletedChecks++
			if result.Status == CheckStatusComplete {
				ts.SuccessfulChecks++
			} else {
				ts.FailedChecks++
			}
			ts.mu.Unlock()
		}(sample)
	}

	wg.Wait()

	ts.mu.Lock()
	ts.Status = CheckStatusComplete
	ts.mu.Unlock()

	log.Infof("Check suite %s completed: %d/%d successful",
		ts.Id, ts.SuccessfulChecks, ts.TotalChecks)
}

func (ts *CheckSuite) testDomain(domain string) CheckResult {
	result := CheckResult{
		Domain:    domain,
		Status:    CheckStatusRunning,
		Timestamp: time.Now(),
	}

	// Check URL - use a common test endpoint
	testURL := fmt.Sprintf("https://%s/", domain)
	if ts.Config.CheckURL != "" {
		testURL = fmt.Sprintf(ts.Config.CheckURL, domain)
	}

	log.Tracef("Testing: %s (timeout: %v)", testURL, ts.Config.Timeout)

	start := time.Now()
	bytesRead, statusCode, err := ts.fetchURL(testURL)
	duration := time.Since(start)

	result.Duration = duration
	result.BytesRead = bytesRead
	result.StatusCode = statusCode

	if err != nil {
		if bytesRead > 0 && statusCode > 0 {
			result.Status = CheckStatusComplete
			if duration.Seconds() > 0 {
				result.Speed = float64(bytesRead) / duration.Seconds()
			}
			log.Tracef("✓ %s: %d bytes, status %d (error ignored: %v)",
				domain, bytesRead, statusCode, err)
		} else {
			result.Status = CheckStatusFailed
			result.Error = err.Error()
			log.Tracef("✗ %s: connection failed: %v", domain, err)
		}
	} else {
		result.Status = CheckStatusComplete
		if duration.Seconds() > 0 {
			result.Speed = float64(bytesRead) / duration.Seconds()
		}
		log.Tracef("✓ %s: %d bytes in %v (%.2f KB/s, status: %d)",
			domain, bytesRead, duration, result.Speed/1024, statusCode)
	}

	return result
}

func (ts *CheckSuite) fetchURL(url string) (int64, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ts.Config.Timeout)
	defer cancel()

	client := &http.Client{
		Timeout: ts.Config.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			ResponseHeaderTimeout: ts.Config.Timeout,
			IdleConnTimeout:       ts.Config.Timeout,
			DialContext: (&net.Dialer{
				Timeout:   ts.Config.Timeout,
				KeepAlive: ts.Config.Timeout,
			}).DialContext,
		},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("request creation failed: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode

	bytesRead, err := io.CopyN(io.Discard, resp.Body, 100*1024)
	if err != nil && err != io.EOF {
		return bytesRead, statusCode, nil
	}

	return bytesRead, statusCode, nil
}

func (ts *CheckSuite) calculateSummary() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if len(ts.Results) == 0 {
		return
	}

	var totalSpeed float64
	var maxSpeed float64
	var minSpeed float64 = -1
	var fastestDomain string
	var slowestDomain string
	successCount := 0

	for _, result := range ts.Results {
		if result.Status != CheckStatusComplete {
			continue
		}

		successCount++
		totalSpeed += result.Speed

		if result.Speed > maxSpeed {
			maxSpeed = result.Speed
			fastestDomain = result.Domain
		}

		if minSpeed < 0 || result.Speed < minSpeed {
			minSpeed = result.Speed
			slowestDomain = result.Domain
		}
	}

	ts.Summary = CheckSummary{
		FastestDomain: fastestDomain,
		SlowestDomain: slowestDomain,
	}

	if successCount > 0 {
		ts.Summary.AverageSpeed = totalSpeed / float64(successCount)
		ts.Summary.SuccessRate = float64(successCount) / float64(len(ts.Results)) * 100
	}
}

func (ts *CheckSuite) GetSnapshot() *CheckSuite {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	snapshot := &CheckSuite{
		Id:                     ts.Id,
		Status:                 ts.Status,
		StartTime:              ts.StartTime,
		EndTime:                ts.EndTime,
		TotalChecks:            ts.TotalChecks,
		CompletedChecks:        ts.CompletedChecks,
		SuccessfulChecks:       ts.SuccessfulChecks,
		FailedChecks:           ts.FailedChecks,
		Summary:                ts.Summary,
		Config:                 ts.Config,
		PresetResults:          ts.PresetResults,
		DomainDiscoveryResults: ts.DomainDiscoveryResults,
		CurrentPhase:           ts.CurrentPhase,
		WorkingFamilies:        ts.WorkingFamilies,
	}

	snapshot.Results = make([]CheckResult, len(ts.Results))
	copy(snapshot.Results, ts.Results)

	return snapshot
}
