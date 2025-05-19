dev:
	make -j3 dev-backend dev-frontend dev-sendgrid-mock

dev-backend:
	cd backend && make dev

dev-frontend:
	cd frontend && npm run dev

dev-sendgrid-mock:
	trap 'docker stop sendgrid-mock' INT TERM; docker start -a sendgrid-mock || docker run -p 3030:3030 --name sendgrid-mock -e SENDGRID_API_KEY=sendgrid -t yudppp/simple-sendgrid-mock-server

migrate: 
	goose --dir ./database/migrations postgres postgresql://postgres:password@localhost:5432/lugia up
initdb:
	psql -U postgres -h localhost -p 5432 -d lugia -f ./database/drop.sql && make migrate