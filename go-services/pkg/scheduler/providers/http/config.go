package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/log"

	imetrics "github.com/KarineAyrs/safe-ai-coder/pkg/metrics"
	"github.com/KarineAyrs/safe-ai-coder/pkg/scheduler/providers/http/client"
	"github.com/KarineAyrs/safe-ai-coder/pkg/server"
)

// ConfigOption configures defaults for bonuses http provider.
type ConfigOption func(*ClientProvider)

// NewProvider creates new Provider instance.
func NewProvider(
	url string,
	options ...ConfigOption,
) (*ClientProvider, error) {
	s := ClientProvider{
		url:          url,
		client:       &client.ClientWithResponses{},
		httpClient:   &http.Client{},
		logger:       log.NewNopLogger(),
		getThrottler: &server.Throttle{},
	}

	for _, opt := range options {
		opt(&s)
	}

	c, err := client.NewClientWithResponses(s.url, []client.ClientOption{
		client.WithBaseURL(s.url),
		client.WithHTTPClient(s.httpClient),
	}...)

	if err != nil {
		return nil, fmt.Errorf("error creating provider for bonuses backend, err: %w", err)
	}
	s.client = c

	return &s, nil
}

// WithHTTPRequestTimeout configures an http request timeout.
func WithHTTPRequestTimeout(httpReqTimeout time.Duration) ConfigOption {
	return func(p *ClientProvider) {
		p.httpClient = &http.Client{
			Timeout: httpReqTimeout,
		}
	}
}

// WithLogger configures service to use specified Logger.
func WithLogger(l log.Logger) ConfigOption {
	return func(p *ClientProvider) {
		p.logger = l
	}
}

// WithResponseMetrics configures a http response metric counter.
func WithResponseMetrics(counter metrics.Counter) ConfigOption {
	return func(p *ClientProvider) {
		p.httpClient.Transport = imetrics.NewMetricsRoundTripper(http.DefaultTransport, counter)
	}
}
