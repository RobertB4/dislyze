dev:
	$(HOME)/go/bin/air

test-unit:
	go test `go list ./... | grep -v test`

test-integration:
	cd test && sh test.sh

sqlc:
	sqlc generate

migrate: 
	goose --dir ../database/migrations postgres postgresql://postgres:password@localhost:5432/lugia up
initdb:
	psql -U postgres -h localhost -p 5432 -d lugia -f ../database/drop.sql && make migrate