FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

RUN apk add --no-cache ca-certificates

RUN addgroup -S unifi && adduser -S -G unifi unifi

ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/go-unifi-mcp /usr/local/bin/go-unifi-mcp

USER unifi

ENTRYPOINT ["/usr/local/bin/go-unifi-mcp"]
