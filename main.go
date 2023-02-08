package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

var kubeSAPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

func main() {
	mux := http.NewServeMux()

	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	conf, err := readConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to open configuration file %q: %s", *configPath, err.Error())
		return
	}

	for name, endpoint := range conf.Endpoints {
		log.Printf("Registering endpoint %q at %s", name, endpoint.Path)

		authToken, err := getAuthToken(endpoint.Auth)
		if err != nil {
			log.Fatalf("Failed to Bearer token: %s", err.Error())
			return
		}

		fetcher := metricsFetcher{
			url: endpoint.Target,
			client: http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: endpoint.InsecureSkipVerify,
					},
				},
			},
			refreshInterval: endpoint.RefreshInterval,
			authToken:       authToken,
		}
		mux.HandleFunc(endpoint.Path, handler(&fetcher))
	}

	srv := &http.Server{
		Addr:         conf.Addr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      mux,
	}
	log.Printf("Listening on %s", conf.Addr)
	log.Println(srv.ListenAndServe())
}

func getAuthToken(conf endpointAuth) (string, error) {
	switch conf.Type {
	case authModeBearer:
		return fmt.Sprintf("Bearer %s", conf.Token), nil
	case authModeKube:
		saToken, err := os.ReadFile(kubeSAPath)
		if err != nil {
			return "", fmt.Errorf("failed to get kubernetes serviceaccount token: %w", err)
		}
		return fmt.Sprintf("Bearer %s", string(saToken)), nil
	case authModeNone:
		return "", nil
	default:
		return "", fmt.Errorf("unkown auth type: %q", conf.Type)
	}
}
