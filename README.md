# Full Stack Monorepo

This monorepo contains a Go backend server and a SvelteKit frontend application.

## Project Structure

```
.
├── backend/         # Go HTTP server
│   ├── main.go     # Server entry point
│   └── go.mod      # Go module file
│
└── frontend/       # SvelteKit application
    ├── src/        # Source code
    ├── static/     # Static assets
    └── package.json # Node.js dependencies
```

## Requirements

### Backend
- Go 1.23 or higher
- golangci-lint (for development)

### Frontend
- Node.js (Latest LTS recommended)
- npm or pnpm

## Development

### Starting the Backend

```bash
cd backend
go run main.go
```

The backend will start on http://localhost:1337

### Starting the Frontend

```bash
cd frontend
npm install    # or pnpm install
npm run dev    # or pnpm dev
```

The frontend will start on http://localhost:1338

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
- Frontend linting: `cd frontend && npm run lint`
- Frontend testing: `cd frontend && npm run test`
- Frontend E2E tests: `cd frontend && npm run test:e2e` 