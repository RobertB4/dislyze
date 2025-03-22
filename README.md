# Full Stack Application

This project consists of a Go backend server and a SvelteKit frontend application.

## Project Structure

- `backend/` - Go HTTP server (port 1337)
- `frontend/` - SvelteKit application (port 1338)

## Backend

### Requirements

- Go 1.23 or higher
- golangci-lint (for development)

### Running the Backend Server

```bash
cd backend
go run main.go
```

The server will start on http://localhost:1337

### Running Linters

```bash
cd backend
golangci-lint run
```

## Frontend

### Requirements

- Node.js (Latest LTS recommended)
- pnpm (recommended) or npm

### Running the Frontend

```bash
cd frontend
pnpm install    # or npm install
pnpm dev       # or npm run dev
```

The frontend will start on http://localhost:1338

### Building for Production

```bash
cd frontend
pnpm build     # or npm run build
``` 