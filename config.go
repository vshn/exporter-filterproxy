package main

import (
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type config struct {
	Addr      string
	Endpoints map[string]endpointConfig
}

type endpointConfig struct {
	Path            string
	Target          string
	RefreshInterval time.Duration
	Auth            endpointAuth
}

type endpointAuth struct {
	Type  authType
	Token string
}

type authType string

var (
	authModeNone   authType = ""
	authModeBearer authType = "Bearer"
	authModeKube   authType = "Kubernetes"
)

func readConfig(path string) (config, error) {
	configFile, err := os.ReadFile(path)
	if err != nil {
		return config{}, err
	}

	conf := config{
		Addr: ":80",
	}
	err = yaml.Unmarshal(configFile, &conf)
	if err != nil {
		return config{}, err
	}
	return conf, nil
}
