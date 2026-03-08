verify: verify-go verify-frontend verify-structural

verify-structural: verify-claude-refs verify-feature-docs

verify-go:
	cd lugia-backend && make lint
	cd giratina-backend && make lint
	cd jirachi && make lint
	cd lugia-backend && make test-unit
	cd giratina-backend && make test-unit
	cd jirachi && make test-unit
	cd lugia-backend && make deadcode
	cd giratina-backend && make deadcode

verify-frontend: build-zoroark
	cd zoroark && npm run check
	cd lugia-frontend && npm run check
	cd giratina-frontend && npm run check
	cd zoroark && npm run lint
	cd lugia-frontend && npm run lint
	cd giratina-frontend && npm run lint

build-zoroark:
	cd zoroark && npm run build

verify-claude-refs:
	@./scripts/verify-claude-refs.sh

verify-feature-docs:
	@./scripts/verify-feature-docs.sh

generate:
	cd lugia-backend && make sqlc
	cd giratina-backend && make sqlc
	cd jirachi && make sqlc
	cd lugia-backend && go run ./cmd/openapi
	cd lugia-frontend && npx openapi-typescript ../lugia-backend/openapi.json --root-types --root-types-no-schema-prefix -o src/schema.ts
	cd giratina-backend && go run ./cmd/openapi
	cd giratina-frontend && npx openapi-typescript ../giratina-backend/openapi.json --root-types --root-types-no-schema-prefix -o src/schema.ts

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
	goose --dir ./database/migrations postgres postgresql://postgres:postgres@localhost:5432/dislyze up
seed:
	PGPASSWORD=postgres psql -U postgres -h localhost -p 5432 -d dislyze -f ./database/seed.sql
initdb:
	PGPASSWORD=postgres psql -U postgres -h localhost -p 5432 -d dislyze -f ./database/drop.sql && make migrate && make seed

devcontainer:
	docker exec -it $$(docker ps -qf "label=devcontainer.local_folder=$$(pwd)") bash

# claude: start a new claude code session
# claudec: claude --continue
# clauder: claude --resume
claude:
	docker exec -it $$(docker ps -qf "label=devcontainer.local_folder=$$(pwd)") claude --dangerously-skip-permissions

claudec:
	docker exec -it $$(docker ps -qf "label=devcontainer.local_folder=$$(pwd)") claude --dangerously-skip-permissions --continue

clauder:
	docker exec -it $$(docker ps -qf "label=devcontainer.local_folder=$$(pwd)") claude --dangerously-skip-permissions --resume
