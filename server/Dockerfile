FROM golang:1.26.2-alpine AS build

WORKDIR /src

COPY VERSION ./VERSION
COPY server/go.mod server/go.sum ./server/

WORKDIR /src/server
RUN go mod download

WORKDIR /src
COPY server ./server

ARG APP_VERSION=
ARG APP_COMMIT=
ARG APP_BUILD_TIME=

WORKDIR /src/server
RUN set -eux; \
    if [ -z "$APP_VERSION" ]; then APP_VERSION="$(cat ../VERSION)"; fi; \
    if [ -z "$APP_BUILD_TIME" ]; then APP_BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"; fi; \
    LDFLAGS="-s -w -X opspilot/server/internal/config.BuildVersion=${APP_VERSION} -X opspilot/server/internal/config.BuildCommit=${APP_COMMIT} -X opspilot/server/internal/config.BuildTime=${APP_BUILD_TIME}"; \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "$LDFLAGS" -o /out/api ./cmd/api; \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "$LDFLAGS" -o /out/agent ./cmd/agent

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 10001 opspilot \
    && mkdir -p /app/.tmp \
    && chown -R opspilot:opspilot /app

WORKDIR /app

COPY --from=build --chown=opspilot:opspilot /out/api /app/api
COPY --from=build --chown=opspilot:opspilot /out/agent /app/agent

USER opspilot

EXPOSE 8080

CMD ["/app/api"]
