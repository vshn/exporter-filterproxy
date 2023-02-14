package target

import (
	"context"
	"crypto/tls"
	"net/http"
	"sync"
	"time"

	dto "github.com/prometheus/client_model/go"
)

type StaticFetcher struct {
	URL       string
	Client    *http.Client
	AuthToken string

	clock           func() time.Time
	refreshInterval time.Duration
	mutex           sync.Mutex
	cache           []dto.MetricFamily
	lastUpdated     time.Time
}

func NewStaticFetcher(url string, authToken string, refreshInterval time.Duration, insecureSkipVerify bool) *StaticFetcher {
	return &StaticFetcher{
		URL: url,
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: insecureSkipVerify,
				},
			},
		},
		refreshInterval: refreshInterval,
		AuthToken:       authToken,
	}
}

// FetchMetrics will fetch and parse the exposed metrics of the configured exporter.
// If a refreshInterval is set the method will cache the response, so if the method is called multiple times in the configured
// refreshInterval interval, only the first call will result in a request to the upstream exporter.
func (f *StaticFetcher) FetchMetrics(_ context.Context) ([]dto.MetricFamily, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.now().Sub(f.lastUpdated) < f.refreshInterval {
		return f.cache, nil
	}

	metrics, err := fetchMetrics(f.Client, f.URL, f.AuthToken)
	if err != nil {
		return nil, err
	}

	f.cache = metrics
	f.lastUpdated = f.now()
	return metrics, nil
}

func (f *StaticFetcher) now() time.Time {
	if f.clock != nil {
		return f.clock()
	}
	return time.Now()
}
