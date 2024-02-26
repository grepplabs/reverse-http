FROM golang:1.22-alpine3.19 AS builder
# hadolint ignore=DL3018
RUN apk add --no-cache alpine-sdk ca-certificates curl

WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN make vendor build

FROM alpine:3.19

COPY --from=builder /app/reverse-http /reverse-http

USER 65532:65532
ENTRYPOINT ["/reverse-http"]
