FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/golang-base ./cmd/api/main.go && \
    CGO_ENABLED=0 GOOS=linux go build -o /app/bin/migrate ./cmd/migrate/main.go && \
    CGO_ENABLED=0 GOOS=linux go build -o /app/bin/seed ./cmd/seed/main.go

FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/bin/golang-base .
COPY --from=builder /app/bin/migrate .
COPY --from=builder /app/bin/seed .
COPY --from=builder /app/migrations ./migrations
EXPOSE 3100
CMD ["./golang-base"]
