package main

import (
	dto "github.com/prometheus/client_model/go"
)

// Filter takes a slice of MetricFamily and returns a slice of MetricFamily that only contains the
// metrics that have *all* labels in the filterLabels.
func Filter(metrics []dto.MetricFamily, filterLabels map[string]string) []dto.MetricFamily {
	res := []dto.MetricFamily{}
	for _, mf := range metrics {
		fmf := filterMetricFamily(mf, filterLabels)
		if len(fmf.Metric) == 0 {
			continue
		}
		res = append(res, *fmf)
	}
	return res
}

func filterMetricFamily(mf dto.MetricFamily, filterLabels map[string]string) *dto.MetricFamily {
	ms := []*dto.Metric{}
	for _, m := range mf.GetMetric() {
		if matchesFilter(m, filterLabels) {
			ms = append(ms, m)
		}
	}
	mf.Metric = ms
	return &mf
}

func matchesFilter(m *dto.Metric, filterLabels map[string]string) bool {
	matched := 0

	for _, l := range m.GetLabel() {
		val, ok := filterLabels[l.GetName()]
		if ok && l.GetValue() != val {
			return false
		} else if ok {
			matched++
		}
	}

	return matched == len(filterLabels)
}
