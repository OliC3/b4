package discovery

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

// DiscoveryPhase represents the current phase of hierarchical discovery
type DiscoveryPhase string

const (
	PhaseBaseline    DiscoveryPhase = "baseline"
	PhaseStrategy    DiscoveryPhase = "strategy_detection"
	PhaseOptimize    DiscoveryPhase = "optimization"
	PhaseCombination DiscoveryPhase = "combination"
)

// StrategyFamily groups related bypass techniques
type StrategyFamily string

const (
	FamilyNone    StrategyFamily = "none"
	FamilyTCPFrag StrategyFamily = "tcp_frag"
	FamilyTLSRec  StrategyFamily = "tls_record"
	FamilyOOB     StrategyFamily = "oob"
	FamilyIPFrag  StrategyFamily = "ip_frag"
	FamilyFakeSNI StrategyFamily = "fake_sni"
	FamilySACK    StrategyFamily = "sack"
	FamilySynFake StrategyFamily = "syn_fake"
)

type CheckResult struct {
	Domain      string            `json:"domain"`
	Category    string            `json:"category"`
	Status      CheckStatus       `json:"status"`
	Duration    time.Duration     `json:"duration"`
	Speed       float64           `json:"speed"`
	BytesRead   int64             `json:"bytes_read"`
	Error       string            `json:"error,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	IsBaseline  bool              `json:"is_baseline"`
	Improvement float64           `json:"improvement"`
	StatusCode  int               `json:"status_code"`
	Set         *config.SetConfig `json:"set"`
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

	// Hierarchical discovery fields
	CurrentPhase    DiscoveryPhase `json:"current_phase,omitempty"`
	WorkingFamilies []string       `json:"working_families,omitempty"`
}

type CheckSummary struct {
	AverageSpeed       float64 `json:"average_speed"`
	AverageImprovement float64 `json:"average_improvement"`
	FastestDomain      string  `json:"fastest_domain"`
	SlowestDomain      string  `json:"slowest_domain"`
	SuccessRate        float64 `json:"success_rate"`
}

type CheckConfig struct {
	CheckURL               string        `json:"check_url"`
	Timeout                time.Duration `json:"timeout"`
	ConfigPropagateTimeout time.Duration `json:"config_propagate_timeout"`
	SamplesPerDomain       int           `json:"samples_per_domain"`
	MaxConcurrent          int           `json:"max_concurrent"`
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
	PresetName string            `json:"preset_name"`
	Family     StrategyFamily    `json:"family,omitempty"`
	Phase      DiscoveryPhase    `json:"phase,omitempty"`
	Status     CheckStatus       `json:"status"`
	Duration   time.Duration     `json:"duration"`
	Speed      float64           `json:"speed"`
	BytesRead  int64             `json:"bytes_read"`
	Error      string            `json:"error,omitempty"`
	StatusCode int               `json:"status_code"`
	Set        *config.SetConfig `json:"set"`
}

type DomainDiscoveryResult struct {
	Domain          string                         `json:"domain"`
	BestPreset      string                         `json:"best_preset"`
	BestSpeed       float64                        `json:"best_speed"`
	BestSuccess     bool                           `json:"best_success"`
	Results         map[string]*DomainPresetResult `json:"results"`
	WorkingFamilies []StrategyFamily               `json:"working_families,omitempty"`
	BaselineSpeed   float64                        `json:"baseline_speed,omitempty"`
	Improvement     float64                        `json:"improvement,omitempty"`
}

// DomainCluster groups domains that likely need the same bypass config
type DomainCluster struct {
	ID             string   `json:"id"`
	Domains        []string `json:"domains"`
	Representative string   `json:"representative"` // Domain we actually test
	BestPreset     string   `json:"best_preset,omitempty"`
	BestSpeed      float64  `json:"best_speed,omitempty"`
	Tested         bool     `json:"tested"`
}

// ConfigPreset represents a bypass configuration to test
type ConfigPreset struct {
	Name                   string           `json:"name"`
	Description            string           `json:"description"`
	Family                 StrategyFamily   `json:"family"`
	Phase                  DiscoveryPhase   `json:"phase"`
	Config                 config.SetConfig `json:"config"`
	Priority               int              `json:"priority"` // Lower = test first
	ConfigPropagateTimeout int              `json:"propagate_timeout"`
}

// StrategyResult tracks whether a strategy family works
type StrategyResult struct {
	Family  StrategyFamily
	Works   bool
	Speed   float64
	Preset  string
	Latency time.Duration
}
