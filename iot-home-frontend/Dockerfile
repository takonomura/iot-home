FROM golang:alpine AS builder

RUN apk add --no-cache git

ENV GO111MODULE=on
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

FROM alpine

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY index.html ./
COPY assets ./assets

COPY --from=builder /app/iot-home-frontend /

ENTRYPOINT ["/iot-home-frontend"]
