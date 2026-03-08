check-structural: verify-claude-refs verify-feature-docs

verify-claude-refs:
	@./scripts/verify-claude-refs.sh

verify-feature-docs:
	@./scripts/verify-feature-docs.sh

periodic-review:
	@echo ""
	@echo "══════════════════════════════════════════════════════════════"
	@echo "  PERIODIC REVIEW — Write a status update"
	@echo "══════════════════════════════════════════════════════════════"
	@echo ""
	@echo "  1. DONE:       What did you complete since your last update?"
	@echo "  2. VERIFIED:   What did you test? (specific commands, tools,"
	@echo "                 playwright — not just 'make check')"
	@echo "  3. NEXT:       What are you about to do? Does it match the plan?"
	@echo "  4. BLOCKED:    Anything stuck or concerning? (skip if nothing)"
	@echo "  5. DISCOVERED: Anything unexpected? (skip if nothing)"
	@echo ""
	@echo "  Write your update now, then continue working."
	@echo "══════════════════════════════════════════════════════════════"
	@echo ""

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
