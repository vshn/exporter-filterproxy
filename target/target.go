package target

import (
	"fmt"
	"io"
	"net/http"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

func fetchMetrics(client *http.Client, url string, authToken string) ([]dto.MetricFamily, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if authToken != "" {
		req.Header.Add("Authorization", authToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 300 {
		res, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("got status code %d and failed to read response: %w", resp.StatusCode, err)

		}
		return nil, fmt.Errorf("got status code %d: %s", resp.StatusCode, string(res))
	}

	return decodeMetrics(resp.Body, expfmt.ResponseFormat(resp.Header))
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
