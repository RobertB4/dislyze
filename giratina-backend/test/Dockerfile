FROM golang:1.24-alpine

WORKDIR /app

# Install wget for healthcheck
RUN apk add --no-cache wget

# Copy go mod and sum files from the giratina-backend directory within the context
COPY giratina-backend/go.mod giratina-backend/go.sum ./

# Copy shared library
COPY jirachi /jirachi

# Download dependencies
RUN go mod download

# Install gotestfmt
RUN go install github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt@latest

# Copy source code of the backend from the giratina-backend directory within the context
COPY giratina-backend/ .

# Copy the database directory from the context root into the image
COPY database /database

# Build the application
RUN go build -o main ./main.go

# Expose port
EXPOSE 14001

# Run the application
CMD ["./main"]