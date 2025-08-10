# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app

# Build zoroark library first
COPY zoroark/ /zoroark/
WORKDIR /zoroark
RUN npm install --no-frozen-lockfile
RUN npm run build

# Copy frontend package files
WORKDIR /app
COPY lugia-frontend/package*.json ./lugia-frontend/
WORKDIR /app/lugia-frontend

# Create symlink to built zoroark library
RUN ln -sf /zoroark ../zoroark

RUN npm ci

# Copy frontend source and build
COPY lugia-frontend/ ./
RUN npm run build

# Stage 2: Build Go backend with embedded frontend
FROM golang:1.24-alpine AS backend-builder

# Set working directory
WORKDIR /app

# Copy all module files for dependency resolution
COPY jirachi/go.mod jirachi/go.sum ./jirachi/
COPY lugia-backend/go.mod lugia-backend/go.sum ./lugia-backend/

# Download dependencies
RUN cd lugia-backend && go mod download

# Copy jirachi source code (shared library)
COPY jirachi/ ./jirachi/

# Copy lugia-backend source code
COPY lugia-backend/ ./lugia-backend/

# Copy database migrations
COPY database/ ./database/

# Copy built frontend files for embedding
COPY --from=frontend-builder /app/lugia-frontend/build ./lugia-backend/frontend_embed

# Build the application
WORKDIR /app/lugia-backend
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:3.22

RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app/lugia-backend

# Copy the binary from builder stage
COPY --from=backend-builder /app/lugia-backend/main .

# Copy database migrations
COPY --from=backend-builder /app/database ../database

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]