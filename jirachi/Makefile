sqlc:
	sqlc generate

test-unit:
	go test `go list ./... | grep -v test` -json -v 2>&1 | gotestfmt -hide=empty-packages,successful-tests