.PHONY: build
build:
	mkdir -p out/build
	go build -o ./out/build ./cmd/...

ADDRESS = http://localhost
PORT = 3000
USERNAME = test
PASSWORD = test
INTERVAL = 10
run-exporter:
	go run ./cmd/exporter -a $(ADDRESS):$(PORT) -u $(USERNAME) -p '$(PASSWORD)' -i $(INTERVAL) --metrics -v 2

run-stub:
	go run ./cmd/raritan-stub --port $(PORT) -u $(USERNAME) -p $(PASSWORD) -v 2

export GO_VERSION = 1.15
export PDU_USERNAME = $(USERNAME)
export PDU_PASSWORD = $(PASSWORD)
export PDU_ADDRESS = $(ADDRESS):$(PORT)
SKAFFOLD_FILE = ./deploy/skaffold.yaml
skaffold-build:
	skaffold build -f $(SKAFFOLD_FILE)
skaffold-run:
	skaffold run --detect-minikube --cache-artifacts=false -f $(SKAFFOLD_FILE)