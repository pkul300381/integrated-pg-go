BIN_DIR := bin
GATEWAY := $(BIN_DIR)/gateway
SIMNET := $(BIN_DIR)/simnet

ENDPOINT ?= 127.0.0.1:5001
ADMIN ?= :8080
ECHO_INTERVAL ?= 15s

.PHONY: build sim gw test vet clean

build: $(GATEWAY) $(SIMNET)

$(GATEWAY):
	go build -o $(GATEWAY) ./cmd/gateway

$(SIMNET):
	go build -o $(SIMNET) ./cmd/simnet

sim: $(SIMNET)
	$(SIMNET) -listen :5001

gw: $(GATEWAY)
	$(GATEWAY) -endpoint $(ENDPOINT) -admin $(ADMIN) -echo-interval $(ECHO_INTERVAL)

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f $(GATEWAY) $(SIMNET)
