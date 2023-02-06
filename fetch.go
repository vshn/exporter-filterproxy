package main

import (
	"io"
	"net/http"
	"sync"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

type metricsFetcher struct {
	url    string
	client http.Client

	refreshInterval time.Duration
	mutex           sync.Mutex
	cache           []dto.MetricFamily
	lastUpdated     time.Time
}

func (f *metricsFetcher) FetchMetrics() ([]dto.MetricFamily, error) {

	f.mutex.Lock()
	defer f.mutex.Unlock()

	if time.Since(f.lastUpdated) < f.refreshInterval {
		return f.cache, nil
	}

	resp, err := f.client.Get(f.url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	metrics, err := decodeMetrics(resp.Body, expfmt.ResponseFormat(resp.Header))
	if err != nil {
		return nil, err
	}

	f.cache = metrics
	f.lastUpdated = time.Now()
	return metrics, nil
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
