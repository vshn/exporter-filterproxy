package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"syscall"

	"github.com/prometheus/common/expfmt"
)

func handler(fetcher *metricsFetcher) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		filterLabels, err := parseURLParams(r.URL.Query())
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}

		metrics, err := fetcher.FetchMetrics()
		if err != nil {
			log.Printf("Failed to fetch metrics: %s", err.Error())
			w.WriteHeader(http.StatusBadGateway)
			return
		}

		enc := expfmt.NewEncoder(w, expfmt.FmtText)
		for _, fm := range Filter(metrics, filterLabels) {
			err := enc.Encode(&fm)
			if err != nil && !errors.Is(err, syscall.EPIPE) {
				log.Printf("Failed to encode: %s", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
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
