build:
	mkdir -p out/build
	go build -o ./out/build ./cmd/...

PORT = 3000
USERNAME = test
PASSWORD = test
INTERVAL = 10
run-exporter:
	go run ./cmd/exporter -a http://localhost:$(PORT) -u $(USERNAME) -p $(PASSWORD) -i $(INTERVAL) --metrics -v 2

run-stub:
	go run ./cmd/raritan-stub --port $(PORT) -u $(USERNAME) -p $(PASSWORD) -v 2