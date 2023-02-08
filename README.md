# Exporter Filterproxy

![Go version](https://img.shields.io/github/go-mod/go-version/vshn/exporter-filterproxy)
[![Version](https://img.shields.io/github/v/release/vshn/exporter-filterproxy)][releases]
[![GitHub downloads](https://img.shields.io/github/downloads/vshn/exporter-filterproxy/total)][releases]

[releases]: https://github.com/vshn/exporter-filterproxy/releases

A proxy that can proxy and filter other Prometheus exporters.




## Try Locally

You can run the `exporter-filterproxy` using docker-compose by running.

```
make docker-compose-up
```

You can now access sample metrics at http://localhost:8082/node and filter these metrics using parameters. 
To only access metrics related to CPU 2, access http://localhost:8082/node?cpu=2.

Further there is a Prometheus at http://localhost:9090/ that only accesses node metrics that are related to CPU 2.
