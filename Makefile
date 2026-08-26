.PHONY: run test test/unit test/acceptance

run:
	go run ./cmd/server

test: test/unit test/acceptance

test/unit:
	go test ./internal/...

test/acceptance:
	go test -tags acceptance ./acceptance/...
