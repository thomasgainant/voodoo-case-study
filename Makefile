.PHONY: run client test test/unit test/acceptance test/load generate

PROTOC_PATH := $(LOCALAPPDATA)/protoc/bin
GOPATH_BIN := $(shell go env GOPATH)/bin

run: generate
	go run ./cmd/server

# Requires a running server (make run)
client:
	go run ./cmd/client

generate:
	PATH="$(PROTOC_PATH):$(GOPATH_BIN):$(PATH)" protoc \
		--proto_path=proto \
		--go_out=gen --go_opt=paths=source_relative \
		--go-grpc_out=gen --go-grpc_opt=paths=source_relative \
		voodoo/v1/voodoo.proto

test: test/unit test/acceptance

test/unit:
	go test ./internal/... ./testclient/...

test/acceptance:
	go test -tags acceptance ./acceptance/...

# Requires a running server (make run). SERVER_ADDR defaults to localhost:8080
test/load:
	k6 run load/load_test.js

