package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

type metricsFetcher struct {
	url       string
	client    *http.Client
	authToken string

	clock           func() time.Time
	refreshInterval time.Duration
	mutex           sync.Mutex
	cache           []dto.MetricFamily
	lastUpdated     time.Time
}

func NewMetricsFetcher(url string, authToken string, refreshInterval time.Duration, insecureSkipVerify bool) *metricsFetcher {
	return &metricsFetcher{
		url: url,
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: insecureSkipVerify,
				},
			},
		},
		refreshInterval: refreshInterval,
		authToken:       authToken,
	}
}

// FetchMetrics will fetch and parse the exposed metrics of the configured exporter.
// If a refreshInterval is set the method will cache the response, so if the method is called multiple times in the configured
// refreshInterval interval, only the first call will result in a request to the upstream exporter.
func (f *metricsFetcher) FetchMetrics() ([]dto.MetricFamily, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.now().Sub(f.lastUpdated) < f.refreshInterval {
		return f.cache, nil
	}

	req, err := http.NewRequest(http.MethodGet, f.url, nil)
	if err != nil {
		return nil, err
	}
	if f.authToken != "" {
		req.Header.Add("Authorization", f.authToken)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 300 {
		res, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("got status code %d and failed to read response: %w", resp.StatusCode, err)

		}
		return nil, fmt.Errorf("got status code %d: %s", resp.StatusCode, string(res))
	}

	metrics, err := decodeMetrics(resp.Body, expfmt.ResponseFormat(resp.Header))
	if err != nil {
		return nil, err
	}

	f.cache = metrics
	f.lastUpdated = f.now()
	return metrics, nil
}

func (f *metricsFetcher) now() time.Time {
	if f.clock != nil {
		return f.clock()
	}
	return time.Now()
}

func decodeMetrics(r io.Reader, format expfmt.Format) ([]dto.MetricFamily, error) {
	dec := expfmt.NewDecoder(r, format)
	metrics := []dto.MetricFamily{}

	for {
		mf := dto.MetricFamily{}
		err := dec.Decode(&mf)
		if err == io.EOF {
			return metrics, nil
		}
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, mf)
	}
}
