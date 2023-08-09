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

run-pool-exporter:
	go run ./cmd/exporter -a $(ADDRESS):$(PORT) -u $(USERNAME) -p '$(PASSWORD)' -i $(INTERVAL) --metrics -v 2

run-pool-stub:
	make -j 5 run-pool-stub-1 run-pool-stub-2 run-pool-stub-3 run-pool-stub-4 run-pool-stub-5

run-pool-stub-1:
	go run ./cmd/raritan-stub --port 3001 -u $(USERNAME) -p $(PASSWORD) --pdu-name pdu01 -v 2

run-pool-stub-2:
	go run ./cmd/raritan-stub --port 3002 -u $(USERNAME) -p $(PASSWORD) --pdu-name pdu02 -v 2

run-pool-stub-3:
	go run ./cmd/raritan-stub --port 3003 -u $(USERNAME) -p $(PASSWORD) --pdu-name pdu03 -v 2

run-pool-stub-4:
	go run ./cmd/raritan-stub --port 3004 -u $(USERNAME) -p $(PASSWORD) --pdu-name pdu04 -v 2

run-pool-stub-5:
	go run ./cmd/raritan-stub --port 3005 -u $(USERNAME) -p $(PASSWORD) --pdu-name pdu05 -v 2

export GO_VERSION = 1.15
export PDU_USERNAME = $(USERNAME)
export PDU_PASSWORD = $(PASSWORD)
export PDU_ADDRESS = $(ADDRESS):$(PORT)
SKAFFOLD_FILE = ./deploy/skaffold.yaml
skaffold-build:
	skaffold build -f $(SKAFFOLD_FILE)
skaffold-run:
	skaffold run --detect-minikube --cache-artifacts=false -f $(SKAFFOLD_FILE)
skaffold-delete:
	skaffold delete --detect-minikube -f $(SKAFFOLD_FILE)