package main

import (
	"flag"
	"log"
	"net/http"
	"time"
)

func main() {
	mux := http.NewServeMux()

	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	conf, err := readConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to open configuration file %q: %s", *configPath, err.Error())
		return
	}

	for _, endpoint := range conf.Endpoints {
		mux.HandleFunc(endpoint.Path, handler(
			&metricsFetcher{
				url:             endpoint.Target,
				refreshInterval: endpoint.RefreshInterval,
			},
		))
	}

	srv := &http.Server{
		Addr:         conf.Addr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      mux,
	}
	log.Println(srv.ListenAndServe())
}
