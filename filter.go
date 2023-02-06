package main

import (
	dto "github.com/prometheus/client_model/go"
)

func Filter(metrics []dto.MetricFamily, filterLabels map[string]string) ([]dto.MetricFamily, error) {
	res := []dto.MetricFamily{}
	for _, mf := range metrics {
		fmf := filterMetricFamily(mf, filterLabels)
		if len(fmf.Metric) == 0 {
			continue
		}
		res = append(res, *fmf)
	}
	return res, nil
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
