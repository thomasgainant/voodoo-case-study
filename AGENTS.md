## Base application

We are working on a Go server application.

This application has to stay basic and is meant to be a demonstration. That means it is important to have a way to run the app and run the tests for it very fast. We want two layers of tests: unit and acceptance tests. They need to be updated every time a featured is added or modified.

## gRPC architecture

The server runs on gRPC. All incoming requests are described as protobuf RPCs in `proto/voodoo/v1/voodoo.proto`, which acts as the gateway layer. Go code is generated from that file into `gen/voodoo/v1/` using `make generate` (requires `protoc` and the `protoc-gen-go`/`protoc-gen-go-grpc` plugins on PATH). Never edit files under `gen/` by hand.

Incoming requests are dispatched by `internal/router`, which embeds `UnimplementedVoodooServiceServer`. Adding a new RPC to the proto and wiring it in the router is the standard way to extend the application. Workers are separate packages called by the router.

### Testing the gRPC connection

Both test layers spin up a real gRPC server on a random local port and connect to it through the shared test client in `testclient/client.go`. No mocks or HTTP helpers are used.

```bash
# Run all tests (unit + acceptance)
make test

# Unit tests only (router logic, server wiring)
make test/unit

# Acceptance tests only (full server boot + real gRPC call)
make test/acceptance
```

To test manually against a running server, use [grpcurl](https://github.com/fullstorydev/grpcurl):

```bash
# Start the server
make run

# Use the client script in cmd\client\main.go
make client
```
