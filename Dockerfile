FROM  golang:1.26-alpine AS builder

WORKDIR /app 

COPY go.mod go.sum ./
RUN go mod download


COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o tandem ./cmd/tandem-cli/main.go



FROM alpine:latest

WORKDIR /root/

RUN apk add bash

COPY --from=builder /app/tandem . 
# CMD ["./tandem"]


