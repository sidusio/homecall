FROM golang:alpine AS builder

WORKDIR /app

COPY go.* .

RUN go mod download

COPY . /app

RUN go build -o out/ ./cmd/...

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root

COPY --from=builder /app/out/ .

CMD ["./homecall"]
