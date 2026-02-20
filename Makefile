verify: verify-go verify-frontend

verify-go:
	cd lugia-backend && make lint
	cd giratina-backend && make lint
	cd jirachi && make lint
	cd lugia-backend && make test-unit
	cd giratina-backend && make test-unit
	cd jirachi && make test-unit

verify-frontend: build-zoroark
	cd zoroark && npm run check
	cd lugia-frontend && npm run check
	cd giratina-frontend && npm run check
	cd zoroark && npm run lint
	cd lugia-frontend && npm run lint
	cd giratina-frontend && npm run lint

build-zoroark:
	cd zoroark && npm run build

generate:
	cd lugia-backend && make sqlc
	cd giratina-backend && make sqlc
	cd jirachi && make sqlc

dev:
	make -j6 dev-lugia-backend dev-lugia-frontend dev-giratina-backend dev-giratina-frontend dev-sendgrid-mock dev-keycloak-mock

dev-lugia-backend:
	cd lugia-backend && make dev

dev-lugia-frontend:
	cd lugia-frontend && npm run dev

dev-giratina-backend:
	cd giratina-backend && make dev

dev-giratina-frontend:
	cd giratina-frontend && npm run dev

dev-sendgrid-mock:
	cd sendgrid-mock && SENDGRID_API_KEY=sendgrid npm run start

dev-keycloak-mock:
	cd keycloak-mock && ./start.sh

migrate: 
	goose --dir ./database/migrations postgres postgresql://postgres:password@localhost:5432/dislyze up
seed:
	psql -U postgres -h localhost -p 5432 -d dislyze -f ./database/seed.sql
initdb:
	psql -U postgres -h localhost -p 5432 -d dislyze -f ./database/drop.sql && make migrate && make seed