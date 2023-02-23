package main

import (
	"context"
	"log"

	"github.com/vshn/exporter-filterproxy/target"
)

type targetConfigFetcher interface {
	FetchTargetConfigs(ctx context.Context, baseTarget string, basePath string) ([]target.StaticConfig, error)
}

type multiTargetConfigFetcher map[string]targetConfigFetcher

func (mf multiTargetConfigFetcher) FetchTargetConfigs(ctx context.Context, baseTarget string, basePath string) ([]target.StaticConfig, error) {

	configs := []target.StaticConfig{}
	for path, f := range mf {
		c, err := f.FetchTargetConfigs(ctx, baseTarget, basePath+path)
		if err != nil {
			log.Printf("Failed to fetch targets: %s", err.Error())
			continue
		}
		configs = append(configs, c...)
	}

	return configs, nil
}
