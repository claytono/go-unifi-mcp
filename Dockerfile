FROM alpine:3.24@sha256:c6b073f8e9e58d881f6ffba6ebf3b9ebd60b9e7d6e8337018c0e68bb5c356873

RUN apk add --no-cache ca-certificates

RUN addgroup -S unifi && adduser -S -G unifi unifi

ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/go-unifi-mcp /usr/local/bin/go-unifi-mcp

USER unifi

ENTRYPOINT ["/usr/local/bin/go-unifi-mcp"]
