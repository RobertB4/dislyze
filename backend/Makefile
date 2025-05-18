dev:
	$(HOME)/go/bin/air

test-unit:
	go test `go list ./... | grep -v test`

test-integration:
	cd test && sh test.sh

sqlc:
	sqlc generate
