sqlc:
	sqlc generate

# check and verify are the same for jirachi (library, no integration tests)
check: lint test-unit

verify: check

lint:
	golangci-lint run --timeout 5m

test-unit:
	go test `go list ./... | grep -v test` -json -v 2>&1 | gotestfmt -hide=empty-packages,successful-tests
