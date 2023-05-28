FROM golang:1.20-buster as builder

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY main.go ./

RUN go build -o server ./main.go


FROM debian:buster-slim

WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/server /app/server
COPY config.toml /app/config.toml

CMD ["/app/server"]