FROM alpine:3.24@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a

RUN apk add --no-cache ca-certificates

RUN addgroup -S unifi && adduser -S -G unifi unifi

ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/go-unifi-mcp /usr/local/bin/go-unifi-mcp

USER unifi

ENTRYPOINT ["/usr/local/bin/go-unifi-mcp"]
