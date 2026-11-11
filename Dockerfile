FROM alpine:3.24@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf

RUN apk add --no-cache ca-certificates

RUN addgroup -S unifi && adduser -S -G unifi unifi

ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/go-unifi-mcp /usr/local/bin/go-unifi-mcp

USER unifi

ENTRYPOINT ["/usr/local/bin/go-unifi-mcp"]
