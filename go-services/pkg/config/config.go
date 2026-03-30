package config

import (
	"time"
)

type API struct {
	// HTTPAddr is TCP address service's API listens on.
	HTTPAddr string `yaml:"http_addr"`
	// RateLimit is maximum allowed requests rate per second.
	RateLimit int `yaml:"rate_limit"`
	// Secret is secret to access API on customers' behalf.
	Secret string `yaml:"secret"`
	// OpsSecret is ops secret.
	OpsSecret string `yaml:"ops_secret"`
	// RequireAuth - defines whether we should use SSO/PSK/... for API requests
	// (at the moment we use it to simplify integration testing for us).
	RequireAuth bool `yaml:"require_auth"`
}

type Ops struct {
	// HTTPAddr is TCP address service's private operations-related API (e.g. exposing Prometheus metrics) listens on.
	HTTPAddr string `yaml:"http_addr"`
}

type ExternalAPI struct {
	URL        string `yaml:"url"`
	AuthUserID string `yaml:"auth_user_id"`
	APISecret  string `yaml:"api_secret"`
	// RequestTimeout is a request timeout for payments connections.
	RequestTimeout time.Duration `yaml:"request_timeout"`
}

type Coder struct {
	CacheDir          string `yaml:"cache_dir"`
	StatsDir          string `yaml:"stats_dir"`
	CodeDir           string `yaml:"code_dir"`
	StatementFilename string `yaml:"statement_filename"`
	ModelName         string `yaml:"model_name"`
	OllamaBaseURL     string `yaml:"ollama_base_url"`
	WithChecker       bool   `yaml:"with_checker"`
}
