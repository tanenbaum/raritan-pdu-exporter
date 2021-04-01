# Raritan PDU Sensors Exporter

PDU sensor scraping for Raritan PX2/PX3 PDU via JSON RPC endpoints.

Provides a prometheus endpoint to export the sensor data.

Raritan RPC docs are here - https://help.raritan.com/json-rpc/pdu/v3.5.0/index.html

## Building

Requires Golang 1.15+.

Run `make build` to build the project.

See `Makefile` for more commands.

## Running Locally

```
$ go run ./cmd/exporter/ --help
Usage:
  exporter [OPTIONS]

Application Options:
  -a, --address=  Address of the PDU JSON RPC endpoint
      --timeout=  Timeout of PDU RPC requests in seconds (default: 10)
  -u, --username= Username for PDU access [$PDU_USERNAME]
  -p, --password= Password for PDU access [$PDU_PASSWORD]
      --metrics   Enable prometheus metrics endpoint
      --port=     Prometheus metrics port (default 2112)
  -i, --interval= Interval between data scrapes (default: 10)

Help Options:
  -h, --help      Show this help message
```

klog flags are also parsed (same flags as glog, see https://github.com/google/glog#setting-flags).

## Kubernetes Deployment

Run the exporter via Helm, see `./deploy/charts/pdu-sensors`.

Or, use Skaffold to build images and deploy directly to a cluster, see `./deploy/skaffold.yaml`.

Set `telegrafSidecar.enabled = true` to run a telegraf sidecar to scrape the prometheus endpoint and push data into Influxdb.

For ease of use the chart can also optionally deploy influxdb and grafana, configured to work directly 
with the telegraf exporter. See `./deploy/charts/pdu-sensors/values.yaml` for more info.

## Logging

Using klog - see expected logging conventions - https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/logging.md