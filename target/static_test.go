package target

import (
	"context"
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
		data, err := os.ReadFile("../testdata/simple")
		require.NoError(t, err)
		_, err = rw.Write(data)
		require.NoError(t, err)
	}))
	defer server.Close()

	f := StaticFetcher{
		URL:    server.URL,
		Client: server.Client(),
	}

	metrics, err := f.FetchMetrics(context.TODO())
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

		data, err := os.ReadFile("../testdata/simple")
		require.NoError(t, err)
		_, err = rw.Write(data)
		require.NoError(t, err)
	}))
	defer server.Close()

	f := NewStaticFetcher(server.URL, "foobar", time.Second, false)
	f.Client = server.Client()

	metrics, err := f.FetchMetrics(context.TODO())
	require.NoError(t, err)
	assert.Len(t, metrics, 2)
}

func TestFetchCache(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		data, err := os.ReadFile("../testdata/simple")
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

	f := StaticFetcher{
		URL:             server.URL,
		Client:          server.Client(),
		clock:           fakeClock,
		refreshInterval: 5 * time.Second,
	}

	metrics, err := f.FetchMetrics(context.TODO())
	require.NoError(t, err)
	assert.Len(t, metrics, 2)
	assert.Equal(t, 1, callCount)

	metrics, err = f.FetchMetrics(context.TODO())
	require.NoError(t, err)
	assert.Len(t, metrics, 2)
	assert.Equal(t, 1, callCount)

	*fakeNow = fakeNow.Add(time.Second)
	metrics, err = f.FetchMetrics(context.TODO())
	require.NoError(t, err)
	assert.Len(t, metrics, 2)
	assert.Equal(t, 1, callCount)

	*fakeNow = fakeNow.Add(5 * time.Second)
	metrics, err = f.FetchMetrics(context.TODO())
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

	f := NewStaticFetcher(server.URL, "", time.Second, false)
	f.Client = server.Client()

	_, err := f.FetchMetrics(context.TODO())
	require.Error(t, err)
}

func TestFetchTargetConfigs(t *testing.T) {

	f := NewStaticFetcher("http://foobar.example.com/buzz", "", time.Second, false)

	tconfs, err := f.FetchTargetConfigs(context.TODO(), "proxy.example.com", "/buzz")
	require.NoError(t, err)
	require.Len(t, tconfs, 1)
	tconf := tconfs[0]
	require.Len(t, tconf.Targets, 1)
	assert.Equal(t, "proxy.example.com", tconf.Targets[0])
	assert.Len(t, tconf.Labels, 2)
	assert.EqualValues(t, "/buzz", tconf.Labels["__metrics_path__"])
	assert.EqualValues(t, "/buzz", tconf.Labels["metrics_path"])
}
