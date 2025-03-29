FROM golang:1.23-alpine

WORKDIR /app

# Install wget for healthcheck
RUN apk add --no-cache wget

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o main ./main.go

# Expose port
EXPOSE 11337

# Run the application
CMD ["./main"] 