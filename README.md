# Full Stack Monorepo

This monorepo contains a Go backend server and a SvelteKit frontend application.

## Project Structure

```
.
├── lugia-backend/         # Go HTTP server
│   ├── main.go     # Server entry point
│   └── go.mod      # Go module file
│
└── lugia-frontend/       # SvelteKit application
    ├── src/        # Source code
    ├── static/     # Static assets
    └── package.json # Node.js dependencies
```

## Requirements

### Backend
- Go 1.24 or higher
- golangci-lint (for development)

### Frontend
- Node.js (Latest LTS recommended)
- npm or pnpm

## Development

### Starting the Backend

```bash
cd lugia-backend
go run main.go
```

The backend will start on http://localhost:3001

### Starting the Frontend

```bash
cd lugia-frontend
npm install    # or pnpm install
npm run dev    # or pnpm dev
```

The frontend will start on http://localhost:3000

## Features

- Backend: Simple Go HTTP server with CORS support
- Frontend: SvelteKit application with:
  - TypeScript support
  - Client-side only rendering (SSR disabled)
  - API integration with the backend
  - Modern UI with responsive design
  - Development tools (ESLint, Prettier, Playwright, Vitest)

## Development Tools

- Go linting: `cd backend && golangci-lint run`
- Frontend linting: `cd lugia-frontend && npm run lint`
- Frontend testing: `cd lugia-frontend && npm run test`
- Frontend E2E tests: `cd lugia-frontend && npm run test:e2e` 