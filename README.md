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
  -n, --name=          Name of the endpoint. Only relevant with multiple endpoints. (default: <hostname>) [$PDU_NAME]
  -a, --address=       Address of the PDU JSON RPC endpoint [$PDU_ADDRESS]
      --timeout=       Timeout of PDU RPC requests in seconds (default: 10)
  -u, --username=      Username for PDU access [$PDU_USERNAME]
  -p, --password=      Password for PDU access [$PDU_PASSWORD]
      --metrics        Enable prometheus metrics endpoint
      --port=          Prometheus metrics port (default: 2112)
  -i, --interval=      Interval between data scrapes (default: 10)
  -c, --config=FILE    path to pool config

Help Options:
  -h, --help           Show this help message
```

klog flags are also parsed (same flags as glog, see https://github.com/google/glog#setting-flags).

### Config file

The exporter is able to export metrics of multiple pdu's. Therefore a json or YAML file is required with the config of each pdu. The config file is provided to the exporter via the `--config` flag. 


    ---
    port: 3000                                    # Listening port 
    metrics: true                                 # Enable prometheus metrics endpoint
    interval: 10                                  # Interval to gather metrics. Exporter will check for new sensors every 10*interval
    username: prometheus                          # username in case no username is defined in pdu_config
    password: supersecure                         # password in case no password is defined in pdu_config
    exporter_labels:
      use_config_name: true                       # Use the name from pdu_config as `pdu_name` label in the metrics. (Defaul: false)
      serial_number: false                        # Add serial number as metric label (Defaul: true)
      snmp_sys_contact: true                      # Add snmp sys_contact as metric label (Defaul: false)
      snmp_sys_name: true                         # Add snmp sys_name as metric label (Defaul: false)
      snmp_sys_location: true                     # Add snmp sys_location as metric label (Defaul: false)
    pdu_config:
      - name: pdu01                               # Name of the PDU endpoint.
        address: "http://pdu01.example.com:3001"  # pdu address
        username: prometheus1                     # pdu username
        password: password01                      # pdu password
      - address: "http://pdu02.example.com:3002"
        username: test
        password: test
      - address: "http://pdu03.example.com:3003"


## Get Metrics

    # single endpoint
    curl http://localhost:2112/metrics

    # multiple endpoints
    curl http://localhost:2112/metrics?endpoint=<pduname>
    curl http://localhost:2112/metrics?endpoint[]=<pduname1>&endpoint[]=<pduname2>
    
    # Wildcard
    curl http://localhost:2112/metrics?endpoint=pdu*


## Stub

    Usage:
      raritan-stub [OPTIONS]

    Application Options:
      -u, --username=    Username for server basic auth [$PDU_USERNAME]
      -p, --password=    Password for server basic auth [$PDU_PASSWORD]
          --port=        Listening port for stub (default: 3000)
          --pdu-outlets= Number of outlets (default: 8) [$PDU_OUTLETS]
          --pdu-inlets=  Number of inlets (default: 2) [$PDU_INLETS]
          --pdu-name=    Name of the pdu (default: Fake Name) [$PDU_NAME]
          --pdu-serial=  Serial of the pdu (default: FAKESERIALNUMBER) [$PDU_SERIAL]

    Help Options:
      -h, --help         Show this help message

When using multiple instances of the stub, you are adviced to set a unique pdu name. 

#### Example

    raritan-stub -u test -p test
    raritan-stub --port 3001 -u test -p test --pdu-outlets 50 --pdu-inlets 4 --pdu-name pdu01 --pdu-serial abcd1234

## Kubernetes Deployment

Run the exporter via Helm, see `./deploy/charts/pdu-sensors`.

Or, use Skaffold to build images and deploy directly to a cluster, see `./deploy/skaffold.yaml`.

Set `telegrafSidecar.enabled = true` to run a telegraf sidecar to scrape the prometheus endpoint and push data into Influxdb.

For ease of use the chart can also optionally deploy influxdb and grafana, configured to work directly 
with the telegraf exporter. See `./deploy/charts/pdu-sensors/values.yaml` for more info.

## Logging

Using klog - see expected logging conventions - https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/logging.md