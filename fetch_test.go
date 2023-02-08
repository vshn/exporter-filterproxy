package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		data, err := os.ReadFile("testdata/simple")
		require.NoError(t, err)
		_, err = rw.Write(data)
		require.NoError(t, err)
	}))
	defer server.Close()

	f := metricsFetcher{
		url:    server.URL,
		client: server.Client(),
	}

	metrics, err := f.FetchMetrics()
	require.NoError(t, err)

	assert.Len(t, metrics, 2)

	// The decoder provided by prometheus doesn't return metric families in a deterministic order
	// so we need to compare them in a order independent way
	mfs := map[string]*dto.MetricFamily{}
	for i, mf := range metrics {
		mfs[mf.GetName()] = &metrics[i]
	}

	assert.Contains(t, mfs, "test_metric_one")
	assert.Len(t, mfs["test_metric_one"].GetMetric(), 3)
	assert.Equal(t, 0.3, mfs["test_metric_one"].GetMetric()[2].Gauge.GetValue())

	assert.Contains(t, mfs, "test_metric_two")
	assert.Len(t, mfs["test_metric_two"].GetMetric(), 3)
	assert.EqualValues(t, 3, mfs["test_metric_two"].GetMetric()[2].Counter.GetValue())
}

func TestFetchAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {

		auth, ok := req.Header["Authorization"]
		require.True(t, ok, "authentication header not set")
		require.Len(t, auth, 1, "multiple authentication headers set")
		require.Equal(t, "foobar", auth[0], "wrong authentication token")

		data, err := os.ReadFile("testdata/simple")
		require.NoError(t, err)
		_, err = rw.Write(data)
		require.NoError(t, err)
	}))
	defer server.Close()

	f := NewMetricsFetcher(server.URL, "foobar", time.Second, false)
	f.client = server.Client()

	metrics, err := f.FetchMetrics()
	require.NoError(t, err)
	assert.Len(t, metrics, 2)
}

func TestFetchCache(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		data, err := os.ReadFile("testdata/simple")
		require.NoError(t, err)
		_, err = rw.Write(data)
		require.NoError(t, err)
		callCount++
	}))
	defer server.Close()

	fakeNow := &time.Time{}
	*fakeNow = time.Now()
	fakeClock := func() time.Time {
		return *fakeNow
	}

	f := metricsFetcher{
		url:             server.URL,
		client:          server.Client(),
		clock:           fakeClock,
		refreshInterval: 5 * time.Second,
	}

	metrics, err := f.FetchMetrics()
	require.NoError(t, err)
	assert.Len(t, metrics, 2)
	assert.Equal(t, 1, callCount)

	metrics, err = f.FetchMetrics()
	require.NoError(t, err)
	assert.Len(t, metrics, 2)
	assert.Equal(t, 1, callCount)

	*fakeNow = fakeNow.Add(time.Second)
	metrics, err = f.FetchMetrics()
	require.NoError(t, err)
	assert.Len(t, metrics, 2)
	assert.Equal(t, 1, callCount)

	*fakeNow = fakeNow.Add(5 * time.Second)
	metrics, err = f.FetchMetrics()
	require.NoError(t, err)
	assert.Len(t, metrics, 2)
	assert.Equal(t, 2, callCount)
}

func TestFetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(418)
		_, err := rw.Write([]byte("I'm a teapot"))
		require.NoError(t, err)
	}))
	defer server.Close()

	f := NewMetricsFetcher(server.URL, "", time.Second, false)
	f.client = server.Client()

	_, err := f.FetchMetrics()
	require.Error(t, err)
}
