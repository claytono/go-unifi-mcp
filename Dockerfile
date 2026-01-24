FROM alpine:3.21

ARG TARGETPLATFORM

RUN apk add --no-cache ca-certificates

COPY go-unifi-mcp /usr/local/bin/go-unifi-mcp

ENTRYPOINT ["/usr/local/bin/go-unifi-mcp"]
