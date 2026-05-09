FROM golang:1.24-alpine AS builder
LABEL "language"="go"

RUN apk add --no-cache gcc musl-dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN mkdir -p ./bin
RUN CGO_ENABLED=1 go build -o ./bin/server ./cmd/api

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /src/bin/server .
COPY --from=builder /src/openapi.yaml .
COPY --from=builder /src/migrations ./migrations

RUN adduser -D -u 1000 appuser && chown -R appuser:appuser /app
USER appuser

EXPOSE 8080
CMD ["./server"]
