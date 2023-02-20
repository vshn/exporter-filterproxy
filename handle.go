package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"syscall"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

type metricsFetcher interface {
	FetchMetrics(ctx context.Context) ([]dto.MetricFamily, error)
}
type multiMetricsFetcher interface {
	FetchMetricsFor(ctx context.Context, endpoint string) ([]dto.MetricFamily, error)
}

func handler(fetcher metricsFetcher) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		filterLabels, err := parseURLParams(r.URL.Query())
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		metrics, err := fetcher.FetchMetrics(r.Context())
		if err != nil {
			log.Printf("Failed to fetch metrics: %s", err.Error())
			w.WriteHeader(http.StatusBadGateway)
			return
		}

		writeMetrics(w, metrics, filterLabels)
	})
}

func multiHandler(prefix string, fetcher multiMetricsFetcher) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		filterLabels, err := parseURLParams(r.URL.Query())
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		endpoint := strings.TrimPrefix(r.URL.Path, prefix)
		endpoint = strings.TrimPrefix(endpoint, "/")

		metrics, err := fetcher.FetchMetricsFor(r.Context(), endpoint)
		if err != nil {
			log.Printf("Failed to fetch metrics: %s", err.Error())
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		if metrics == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		writeMetrics(w, metrics, filterLabels)
	})
}

func serviceDiscoveryHandler(prefix string, fetcher targetConfigFetcher) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		configs, err := fetcher.FetchTargetConfigs(r.Context(), r.Host, strings.TrimSuffix(r.URL.Path, "/"))
		if err != nil {
			log.Printf("Failed to discover Endpoints: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := json.Marshal(configs)
		if err != nil {
			log.Printf("Failed to marshal Endpoints: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(res)

		if err != nil {
			log.Printf("Failed to write: %s", err.Error())
			return
		}
	})
}

func parseURLParams(values url.Values) (map[string]string, error) {
	res := map[string]string{}
	for k, v := range values {
		if len(v) != 1 {
			return nil, fmt.Errorf("invalid URL paramters")
		}
		res[k] = v[0]
	}
	return res, nil
}

func writeMetrics(w http.ResponseWriter, metrics []dto.MetricFamily, filterLabels map[string]string) {
	enc := expfmt.NewEncoder(w, expfmt.FmtText)
	for _, fm := range Filter(metrics, filterLabels) {
		err := enc.Encode(&fm)
		if err != nil && !errors.Is(err, syscall.EPIPE) {
			log.Printf("Failed to encode: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
