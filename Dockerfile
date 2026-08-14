FROM golang:1.22-alpine AS builder

WORKDIR /app

# Using pure Go SQLite (github.com/glebarez/sqlite) allows us to disable CGO
# This results in a much smaller and more portable binary
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /meshery-ai-adapter .

FROM alpine:3.19

WORKDIR /app
COPY --from=builder /meshery-ai-adapter .

EXPOSE 9082
CMD ["/app/meshery-ai-adapter"]
