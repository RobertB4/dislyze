dev:
	make -j5 dev-lugia-backend dev-lugia-frontend dev-giratina-backend dev-giratina-frontend dev-sendgrid-mock

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

migrate: 
	goose --dir ./database/migrations postgres postgresql://postgres:password@localhost:5432/dislyze up
seed:
	psql -U postgres -h localhost -p 5432 -d dislyze -f ./database/seed.sql
initdb:
	psql -U postgres -h localhost -p 5432 -d dislyze -f ./database/drop.sql && make migrate && make seed