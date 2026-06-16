FROM alpine:3.24@sha256:f5064d3e5f88c467c714509f491853ab2d951932c5cad699c0cb969dcec6f3b4

RUN apk add --no-cache ca-certificates

RUN addgroup -S unifi && adduser -S -G unifi unifi

ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/go-unifi-mcp /usr/local/bin/go-unifi-mcp

USER unifi

ENTRYPOINT ["/usr/local/bin/go-unifi-mcp"]
