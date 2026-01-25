FROM golang:1.22-alpine AS builder
LABEL "language"="go"

WORKDIR /src
COPY . .
RUN mkdir -p ./bin
RUN go build -o ./bin/server ./cmd/api

FROM alpine:latest
WORKDIR /app
COPY --from=builder /src/bin/server .
COPY --from=builder /src/openapi.yaml .

EXPOSE 8080
CMD ["./server"]
