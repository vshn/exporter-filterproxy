package main

import (
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestFilter(t *testing.T) {

	tcs := map[string]struct {
		input        []dto.MetricFamily
		filterLabels map[string]string
		output       []dto.MetricFamily
	}{
		"EmptyNoFilter": {
			input:  []dto.MetricFamily{},
			output: []dto.MetricFamily{},
		},
		"Empty": {
			input:  []dto.MetricFamily{},
			output: []dto.MetricFamily{},
			filterLabels: map[string]string{
				"foo": "bar",
			},
		},

		"EmptyMF": {
			input: []dto.MetricFamily{
				testMF("empty"),
			},
			output: []dto.MetricFamily{},
			filterLabels: map[string]string{
				"foo": "bar",
			},
		},
		"OneFilter": {
			input: []dto.MetricFamily{
				testMF("one",
					testCounter(1, "foo", "bar"),
					testCounter(3, "foo", "bar", "other", "label"),
					testCounter(2, "foo", "buzz"),
				),
			},
			output: []dto.MetricFamily{
				testMF("one",
					testCounter(1, "foo", "bar"),
					testCounter(3, "foo", "bar", "other", "label"),
				),
			},
			filterLabels: map[string]string{
				"foo": "bar",
			},
		},
		"TwoFilter": {
			input: []dto.MetricFamily{
				testMF("two",
					testCounter(1, "foo", "bar"),
					testCounter(3, "foo", "bar", "other", "label"),
					testCounter(2, "foo", "buzz"),
				),
			},
			output: []dto.MetricFamily{
				testMF("two",
					testCounter(3, "foo", "bar", "other", "label"),
				),
			},
			filterLabels: map[string]string{
				"foo":   "bar",
				"other": "label",
			},
		},
		"MultiMF": {
			input: []dto.MetricFamily{
				testMF("one",
					testCounter(1, "foo", "bar"),
					testCounter(3, "foo", "bar", "other", "label"),
					testCounter(2, "foo", "buzz"),
				),
				testMF("two",
					testGauge(1.1, "l1", "1", "l2", "2"),
					testGauge(1.2, "l1", "1", "l2", "2"),
				),
				testMF("three",
					testGauge(100, "foo", "bar"),
				),
				testMF("four",
					testCounter(1, "foo", "bar"),
					testCounter(1.2, "l1", "1", "l2", "2"),
				),
			},
			output: []dto.MetricFamily{
				testMF("one",
					testCounter(1, "foo", "bar"),
					testCounter(3, "foo", "bar", "other", "label"),
				),
				testMF("three",
					testGauge(100, "foo", "bar"),
				),
				testMF("four",
					testCounter(1, "foo", "bar"),
				),
			},
			filterLabels: map[string]string{
				"foo": "bar",
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.output, Filter(tc.input, tc.filterLabels))
		})
	}
}

func testMF(name string, metrics ...*dto.Metric) dto.MetricFamily {
	return dto.MetricFamily{
		Name:   &name,
		Metric: metrics,
	}
}

func testCounter(value float64, labels ...string) *dto.Metric {
	return &dto.Metric{
		Label: labelPairs(labels),
		Counter: &dto.Counter{
			Value: &value,
		},
	}
}

func testGauge(value float64, labels ...string) *dto.Metric {
	return &dto.Metric{
		Label: labelPairs(labels),
		Gauge: &dto.Gauge{
			Value: &value,
		},
	}
}

func labelPairs(labels []string) []*dto.LabelPair {
	lpair := []*dto.LabelPair{}

	i := 0
	for i < len(labels) {
		key := labels[i]
		val := labels[i+1]
		lpair = append(lpair, &dto.LabelPair{
			Name:  &key,
			Value: &val,
		})
		i = i + 2
	}
	return lpair
}
