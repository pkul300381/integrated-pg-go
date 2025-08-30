# Go Payment Gateway — Pure Stdlib

## Build
```
go build -o bin/gateway ./cmd/gateway
go build -o bin/simnet  ./cmd/simnet
```

## Run local test
```
./bin/simnet -listen :5001
./bin/gateway -endpoint 127.0.0.1:5001 -admin :8080 -echo-interval 15s
```
