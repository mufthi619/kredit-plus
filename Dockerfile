FROM golang:1.22.5-alpine AS builder

# Add necessary build tools and timezone data
RUN apk add --no-cache tzdata make git

# Set necessary environment variables
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Set working directory
WORKDIR /build

# Copy dependency files
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the entire project
COPY . .

# Build the application
RUN go build -o main ./cmd/main.go

# Create distribution stage
WORKDIR /dist
RUN cp /build/main .

# Final stage
FROM alpine:3

# Install necessary runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder stage
COPY --from=builder /dist/main /

# Copy configuration file
COPY ./config/config.yaml /config.yaml

# Set environment variables
ENV TZ=Asia/Jakarta \
    MODE=dev

# Expose port (adjust if needed)
EXPOSE 8080

# Command to run the application
ENTRYPOINT ["/main"]
CMD ["server"]