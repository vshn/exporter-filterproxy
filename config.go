package main

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type config struct {
	Addr      string                    `yaml:"addr"`
	Endpoints map[string]endpointConfig `yaml:"endpoints"`
}

type endpointConfig struct {
	Path               string        `yaml:"path"`
	Target             string        `yaml:"target"`
	KubernetesTarget   *kubeTarget   `yaml:"kubernetes_target"`
	RefreshInterval    time.Duration `yaml:"refresh_interval"`
	Auth               endpointAuth  `yaml:"auth"`
	InsecureSkipVerify bool          `yaml:"insecure_skip_verify"`
}

type kubeTarget struct {
	Endpoint kubeEndpointTarget `yaml:"endpoint"`
}
type kubeEndpointTarget struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
	Port      int    `yaml:"port"`
	Path      string `yaml:"path"`
	Scheme    string `yaml:"scheme"`
}

type endpointAuth struct {
	Type  authType `yaml:"type"`
	Token string   `yaml:"token"`
}

type authType string

var (
	authModeNone   authType = ""
	authModeBearer authType = "Bearer"
	authModeKube   authType = "Kubernetes"
)

func readConfig(path string) (config, error) {
	conf := config{
		Addr: ":80",
	}

	if path == "" {
		return conf, nil
	}
	configFile, err := os.ReadFile(path)
	if err != nil {
		return config{}, err
	}

	err = yaml.Unmarshal(configFile, &conf)
	if err != nil {
		return config{}, err
	}
	return conf, nil
}
