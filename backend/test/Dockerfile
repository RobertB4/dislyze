FROM golang:1.23-alpine

WORKDIR /app

# Install wget for healthcheck
RUN apk add --no-cache wget

# Copy go mod and sum files from the backend directory within the context
COPY backend/go.mod backend/go.sum ./

# Download dependencies
RUN go mod download

# Copy source code of the backend from the backend directory within the context
COPY backend/ .

# Copy the database directory from the context root into the image
COPY database /database

# Build the application
RUN go build -o main ./main.go

# Expose port
EXPOSE 11337

# Run the application
CMD ["./main"] 