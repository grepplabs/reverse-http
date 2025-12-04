FROM alpine:3.23
RUN apk add ca-certificates
COPY reverse-http /
USER 65532:65532
ENTRYPOINT ["/reverse-http"]
