# Multi-stage build for NeuroNetes controllers
FROM golang:1.21-alpine AS builder

# Install dependencies
RUN apk add --no-cache git make

WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the manager
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager cmd/manager/main.go

# Build the scheduler
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o scheduler cmd/scheduler/main.go

# Build the autoscaler
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o autoscaler cmd/autoscaler/main.go

# Use distroless as minimal base image
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
COPY --from=builder /workspace/scheduler .
COPY --from=builder /workspace/autoscaler .

USER 65532:65532

ENTRYPOINT ["/manager"]
