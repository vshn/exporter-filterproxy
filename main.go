package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/vshn/exporter-filterproxy/target"
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

	targetDiscovery := multiTargetConfigFetcher{}

	for name, endpoint := range conf.Endpoints {

		authToken, err := getAuthToken(endpoint.Auth)
		if err != nil {
			log.Fatalf("Failed to get bearer token: %s", err.Error())
			return
		}

		switch {
		case endpoint.Target != "":
			log.Printf("Registering static endpoint %q at %s", name, endpoint.Path)
			sf := target.NewStaticFetcher(endpoint.Target, authToken, endpoint.RefreshInterval, endpoint.InsecureSkipVerify)
			mux.HandleFunc(endpoint.Path,
				handler(sf),
			)
			targetDiscovery[endpoint.Path] = sf
		case endpoint.KubernetesTarget != nil:
			log.Printf("Registering kube endpoint %q at %s", name, endpoint.Path)
			kf, err := target.NewKubernetesEndpointFetcher(
				target.KubernetesEndpointFetcherOpts{
					Endpointname:       endpoint.KubernetesTarget.Endpoint.Name,
					Namespace:          endpoint.KubernetesTarget.Endpoint.Namespace,
					Port:               endpoint.KubernetesTarget.Endpoint.Port,
					Path:               endpoint.KubernetesTarget.Endpoint.Path,
					Scheme:             endpoint.KubernetesTarget.Endpoint.Scheme,
					AuthToken:          authToken,
					RefreshInterval:    endpoint.RefreshInterval,
					InsecureSkipVerify: endpoint.InsecureSkipVerify,
				},
			)
			if err != nil {
				log.Fatalf("Failed to initalize Kubernetes endpoint: %s", err.Error())
				return
			}
			mux.HandleFunc(endpoint.Path+"/",
				multiHandler(endpoint.Path, kf),
			)
			mux.HandleFunc(endpoint.Path,
				serviceDiscoveryHandler(endpoint.Path, kf),
			)
			targetDiscovery[endpoint.Path] = kf
		default:
			log.Fatalf("No target set for endpoint %s", name)
			return
		}

	}

	mux.HandleFunc("/",
		serviceDiscoveryHandler("", targetDiscovery),
	)

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
