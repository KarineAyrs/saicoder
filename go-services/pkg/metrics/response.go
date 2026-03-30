package metrics

import (
	"net/http"
	"strconv"

	"github.com/go-kit/kit/metrics"
)

// RoundTripper type implements the http.RoundTripper interface
// To intercept http client response and collect metrics.
type RoundTripper struct {
	Proxy           http.RoundTripper
	responseCounter metrics.Counter
}

// NewMetricsRoundTripper creates a new instance of RoundTripper.
func NewMetricsRoundTripper(proxy http.RoundTripper, responseCounter metrics.Counter) RoundTripper {
	return RoundTripper{
		Proxy:           proxy,
		responseCounter: responseCounter,
	}
}

// RoundTrip executes a single HTTP transaction, returning
// a Response for the provided Request.
func (rt RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Send the request, get the response (or the error)
	res, e := rt.Proxy.RoundTrip(req)

	if res != nil {
		labels := []string{
			"status_code", strconv.Itoa(res.StatusCode),
		}
		rt.responseCounter.With(labels...).Add(1)
	}

	return res, e
}
