package checker

import (
	"sync"
	"time"

	"github.com/daniellavrushin/b4/config"
)

type CheckStatus string

const (
	CheckStatusPending  CheckStatus = "pending"
	CheckStatusRunning  CheckStatus = "running"
	CheckStatusComplete CheckStatus = "complete"
	CheckStatusFailed   CheckStatus = "failed"
	CheckStatusCanceled CheckStatus = "canceled"
)

type CheckResult struct {
	Domain      string        `json:"domain"`
	Category    string        `json:"category"`
	Status      CheckStatus   `json:"status"`
	Duration    time.Duration `json:"duration"`
	Speed       float64       `json:"speed"`
	BytesRead   int64         `json:"bytes_read"`
	Error       string        `json:"error,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
	IsBaseline  bool          `json:"is_baseline"`
	Improvement float64       `json:"improvement"`
	StatusCode  int           `json:"status_code"`
}

type CheckSuite struct {
	Id                     string                            `json:"id"`
	Status                 CheckStatus                       `json:"status"`
	StartTime              time.Time                         `json:"start_time"`
	EndTime                time.Time                         `json:"end_time"`
	TotalChecks            int                               `json:"total_checks"`
	CompletedChecks        int                               `json:"completed_checks"`
	SuccessfulChecks       int                               `json:"successful_checks"`
	FailedChecks           int                               `json:"failed_checks"`
	Results                []CheckResult                     `json:"results"`
	Summary                CheckSummary                      `json:"summary"`
	PresetResults          map[string]*CheckSummary          `json:"preset_results,omitempty"`
	DomainDiscoveryResults map[string]*DomainDiscoveryResult `json:"domain_discovery_results,omitempty"`
	mu                     sync.RWMutex                      `json:"-"`
	cancel                 chan struct{}                     `json:"-"`
	Config                 CheckConfig                       `json:"config"`
}

type CheckSummary struct {
	AverageSpeed       float64 `json:"average_speed"`
	AverageImprovement float64 `json:"average_improvement"`
	FastestDomain      string  `json:"fastest_domain"`
	SlowestDomain      string  `json:"slowest_domain"`
	SuccessRate        float64 `json:"success_rate"`
}

type CheckConfig struct {
	CheckURL         string        `json:"check_url"`
	Timeout          time.Duration `json:"timeout"`
	SamplesPerDomain int           `json:"samples_per_domain"`
	MaxConcurrent    int           `json:"max_concurrent"`
}

type DomainSample struct {
	Domain   string
	Category string
}

type ConfigTestMode struct {
	Enabled        bool
	OriginalConfig *config.Config
	Presets        []ConfigPreset
	PresetResults  map[string][]CheckResult
}

type DomainPresetResult struct {
	PresetName string        `json:"preset_name"`
	Status     CheckStatus   `json:"status"`
	Duration   time.Duration `json:"duration"`
	Speed      float64       `json:"speed"`
	BytesRead  int64         `json:"bytes_read"`
	Error      string        `json:"error,omitempty"`
	StatusCode int           `json:"status_code"`
}

type DomainDiscoveryResult struct {
	Domain      string                         `json:"domain"`
	BestPreset  string                         `json:"best_preset"`
	BestSpeed   float64                        `json:"best_speed"`
	BestSuccess bool                           `json:"best_success"`
	Results     map[string]*DomainPresetResult `json:"results"`
}
