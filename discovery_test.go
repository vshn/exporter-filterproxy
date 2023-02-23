package main

import (
	"context"
	"testing"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vshn/exporter-filterproxy/target"
)

func TestMultiTargetConfigFetcher(t *testing.T) {

	fa := fakeTargetConfigFetcher{
		t:      t,
		path:   "/a",
		target: "proxy.example.com",
		configs: []target.StaticConfig{
			{
				Targets: []string{"proxy.example.com"},
				Labels: map[model.LabelName]model.LabelValue{
					"foo": "a1",
				},
			},
			{
				Targets: []string{"proxy.example.com"},
				Labels: map[model.LabelName]model.LabelValue{
					"foo": "a2",
				},
			},
		},
	}
	fb := fakeTargetConfigFetcher{
		t:      t,
		path:   "/b",
		target: "proxy.example.com",
		configs: []target.StaticConfig{
			{
				Targets: []string{"proxy.example.com"},
				Labels: map[model.LabelName]model.LabelValue{
					"foo": "b1",
				},
			},
			{
				Targets: []string{"proxy.example.com"},
				Labels: map[model.LabelName]model.LabelValue{
					"foo": "b2",
				},
			},
		},
	}

	mf := multiTargetConfigFetcher{
		"/a": fa,
		"/b": fb,
	}

	conf, err := mf.FetchTargetConfigs(context.TODO(), "proxy.example.com", "")
	require.NoError(t, err)
	require.Len(t, conf, 4)

	seenFoolabels := []string{}
	for _, c := range conf {
		require.Len(t, c.Targets, 1)
		assert.Equal(t, "proxy.example.com", c.Targets[0])
		require.Len(t, c.Labels, 1)

		seenFoolabels = append(seenFoolabels, string(c.Labels["foo"]))
	}
	assert.ElementsMatch(t, seenFoolabels, []string{"b1", "b2", "a1", "a2"})

}

type fakeTargetConfigFetcher struct {
	configs []target.StaticConfig
	t       *testing.T
	path    string
	target  string
}

func (f fakeTargetConfigFetcher) FetchTargetConfigs(ctx context.Context, baseTarget string, basePath string) ([]target.StaticConfig, error) {
	assert.Equal(f.t, f.path, basePath)
	assert.Equal(f.t, f.target, baseTarget)
	return f.configs, nil
}
