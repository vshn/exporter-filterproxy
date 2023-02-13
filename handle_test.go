package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
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
