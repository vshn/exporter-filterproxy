package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vshn/exporter-filterproxy/target"
)

func TestHandler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		data, err := os.ReadFile("testdata/simple")
		require.NoError(t, err)
		_, err = rw.Write(data)
		require.NoError(t, err)
	}))
	defer server.Close()

	f := target.StaticFetcher{
		URL:    server.URL,
		Client: server.Client(),
	}
	h := handler(&f)

	req, err := http.NewRequest("GET", "/metrics?foo=buzz", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, expectedHandlerRes, rr.Body.String())

}

var expectedHandlerRes = `# HELP test_metric_one First sample metric
# TYPE test_metric_one gauge
test_metric_one{foo="buzz"} 0.2
`

func TestMultiHandler(t *testing.T) {

	h := multiHandler("/test", fakeMultiMetricsFetcher{
		"foo": []dto.MetricFamily{
			{
				Name: deref("foo"),
				Help: deref("Foo"),
				Type: deref(dto.MetricType_GAUGE),
				Metric: []*dto.Metric{
					{
						Label: []*dto.LabelPair{
							{
								Name:  deref("foo"),
								Value: deref("bar"),
							},
						},
						Gauge: &dto.Gauge{
							Value: deref(float64(2)),
						},
					},
					{
						Label: []*dto.LabelPair{
							{
								Name:  deref("foo"),
								Value: deref("buzz"),
							},
						},
						Gauge: &dto.Gauge{
							Value: deref(float64(3)),
						},
					},
				},
			},
		},
		"bar": []dto.MetricFamily{
			{
				Name: deref("bar"),
				Help: deref("BAR"),
				Type: deref(dto.MetricType_COUNTER),
				Metric: []*dto.Metric{
					{
						Label: []*dto.LabelPair{
							{
								Name:  deref("foo"),
								Value: deref("bar"),
							},
						},
						Counter: &dto.Counter{
							Value: deref(float64(42)),
						},
					},
					{
						Label: []*dto.LabelPair{
							{
								Name:  deref("bar"),
								Value: deref("buzz"),
							},
						},
						Counter: &dto.Counter{
							Value: deref(float64(3)),
						},
					},
				},
			},
		},
	})

	req, err := http.NewRequest("GET", "/test/foo?foo=buzz", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, expectedMulitHandlerFoo, rr.Body.String())

	req, err = http.NewRequest("GET", "/test/bar?foo=bar", nil)
	require.NoError(t, err)
	rr = httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, expectedMulitHandlerBar, rr.Body.String())

	req, err = http.NewRequest("GET", "/test/buzz", nil)
	require.NoError(t, err)
	rr = httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

var expectedMulitHandlerFoo = `# HELP foo Foo
# TYPE foo gauge
foo{foo="buzz"} 3
`
var expectedMulitHandlerBar = `# HELP bar BAR
# TYPE bar counter
bar{foo="bar"} 42
`

type fakeMultiMetricsFetcher map[string][]dto.MetricFamily

func (f fakeMultiMetricsFetcher) FetchMetricsFor(ctx context.Context, endpoint string) ([]dto.MetricFamily, error) {
	return f[endpoint], nil
}

func deref[T any](x T) *T {
	return &x
}

func TestDiscoveryHandler(t *testing.T) {

	f := fakeTargetConfigFetcher{
		configs: []target.StaticConfig{
			{
				Targets: []string{"proxy.example.com"},
				Labels: map[model.LabelName]model.LabelValue{
					"__metrics_path__": "/foo/a.b.c.d",
					"metrics_path":     "/foo",
					"instance":         "a.b.c.d",
				},
			},
			{
				Targets: []string{"proxy.example.com"},
				Labels: map[model.LabelName]model.LabelValue{
					"__metrics_path__": "/foo/d.e.f.g",
					"metrics_path":     "/foo",
					"instance":         "d.e.f.g",
				},
			},
			{
				Targets: []string{"proxy.example.com"},
				Labels: map[model.LabelName]model.LabelValue{
					"__metrics_path__": "/bar",
					"metrics_path":     "/bar",
				},
			},
		},
		t:      t,
		path:   "",
		target: "proxy.example.com",
	}
	h := serviceDiscoveryHandler("", f)

	req, err := http.NewRequest("GET", "/", nil)
	req.Host = "proxy.example.com"
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	assert.Equal(t, expectedDiscoveryRes, rr.Body.String())
}

var expectedDiscoveryRes = `[{"targets":["proxy.example.com"],"labels":{"__metrics_path__":"/foo/a.b.c.d","instance":"a.b.c.d","metrics_path":"/foo"}},{"targets":["proxy.example.com"],"labels":{"__metrics_path__":"/foo/d.e.f.g","instance":"d.e.f.g","metrics_path":"/foo"}},{"targets":["proxy.example.com"],"labels":{"__metrics_path__":"/bar","metrics_path":"/bar"}}]`
