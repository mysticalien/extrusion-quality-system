FROM golang:1.26.1-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

RUN go install github.com/pressly/goose/v3/cmd/goose@latest

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/simulator ./cmd/simulator


FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/server /app/server
COPY --from=builder /out/simulator /app/simulator
COPY --from=builder /go/bin/goose /usr/local/bin/goose

COPY web /app/web
COPY migrations /app/migrations

EXPOSE 8080

CMD ["/app/server"]