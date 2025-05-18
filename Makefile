dev:
	make -j3 dev-backend dev-frontend dev-sendgrid-mock

dev-backend:
	cd backend && make dev

dev-frontend:
	cd frontend && npm run dev

dev-sendgrid-mock:
	docker run -p 3030:3030 -e SENDGRID_API_KEY=sendgrid -t yudppp/simple-sendgrid-mock-server
