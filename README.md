# Exporter Filterproxy

![Go version](https://img.shields.io/github/go-mod/go-version/vshn/exporter-filterproxy)
[![Version](https://img.shields.io/github/v/release/vshn/exporter-filterproxy)][releases]
[![GitHub downloads](https://img.shields.io/github/downloads/vshn/exporter-filterproxy/total)][releases]

[releases]: https://github.com/vshn/exporter-filterproxy/releases

A proxy that can proxy and filter other Prometheus exporters.
It is intended to expose a subset of metrics of an exporter, for example to expose kube-state-metrics that only relate to a specific namespace.


## Quick Start

You can run the `exporter-filterproxy` using docker-compose or in kubernetes using kind.

### Docker Compose

To test the `exporter-filterproxy` in docker and exposing static targets you can run:

```
make docker-compose-up
```

You can now access sample metrics at http://localhost:8082/node and filter these metrics using parameters. 
To only access metrics related to CPU 2, access http://localhost:8082/node?cpu=2.

Further there is a Prometheus at http://localhost:9090/ that only accesses node metrics that are related to CPU 2.


### KinD

To test the `exporter-filterproxy` in Kubernetes using the Kubernetes service discovery you can run:

```
make kind
```

This will start a local kind cluster and deploy a `exporter-filterproxy` that proxies the metrics of the CoreDNS service.


## Configuration

The filterproxy is configured through a YAML file, where you can configure one or more upstream endpoints of Prometheus exporters.

| Field | Description |
|---|---|
| `addr` | On what address the filterproxy will listen on |
| `endpoints` | A map of upstream Prometheus exporters that will be proxied |
| `endpoints.<exporter>.path` | On what path the exporter `<exporter>` will be proxied |
| `endpoints.<exporter>.target` | The address where to query the exporter `<exporter>` exposes metrics |
| `endpoints.<exporter>.kubernetes_target` | Configuration to expose a Kubernetes service |
| `endpoints.<exporter>.kubernetes_target.name` | The name of the Kubernetes service |
| `endpoints.<exporter>.kubernetes_target.namespace` | The namespace of the Kubernetes service |
| `endpoints.<exporter>.kubernetes_target.port` | The port on which metrics are exposed on |
| `endpoints.<exporter>.kubernetes_target.path` | The path the exporter exposes the metrics on |
| `endpoints.<exporter>.kubernetes_target.scheme` | What scheme the exporter uses to expose metrics (`http` or `https`) |
| `endpoints.<exporter>.refresh_interval` | If set the proxy will only refresh the metrics every refresh interval instead of forwarding every request |
| `endpoints.<exporter>.insecure_skip_verify` | Whether the proxy should skip verifying the exporters certificate |
| `endpoints.<exporter>.auth` | How to authenticate to the exporter. This either has `type: Bearer` and the bearer token needs to be specified in the `token` field, or it can have `type: Kubernetes`, in which case the proxy will authenticate using the service account of the pod it is running in (will only work when running in Kubernetes) |


The following example configuration will run the filterproxy on port `8082`.
It will expose a kube-state-metrics exporter running at `kube.example.com` on at the path `kube-state-metrics` and will authenticate to it by putting the bearer token `foobar` in the authorization header.
The TLS certificate will not be verified and the metrics will be refreshed every 5 seconds.

It will also expose node-exporter metrics running at `node.example.com` at the path `/node`.

```yaml
addr: :8082
endpoints:
  kube_state_metrics:
    path: /kube-state-metrics
    target: https://kube.example.com:8077/metrics
    refresh_interval: 5s
    insecure_skip_verify: true
    auth:
      type: Bearer
      token: foobar
  node:
    path: /node
    target: http://node.example.com:9100/metrics
    refresh_interval: 7s
```


## Development

### Run tests

To run tests simply run

```
make test
```
