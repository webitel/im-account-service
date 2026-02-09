#syntax=docker/dockerfile:1

ARG GO_VERSION=1.25.5
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS build
WORKDIR /src

RUN --mount=type=cache,target=/go/pkg/mod/ \
  --mount=type=bind,source=go.sum,target=go.sum \
  --mount=type=bind,source=go.mod,target=go.mod \
  go mod download -x

ARG TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=bind,target=. \
  CGO_ENABLED=0 GOARCH=${TARGETARCH} go build \
  -trimpath \
  -o /bin/im-account-service main.go

FROM alpine:3.20 AS final


LABEL org.opencontainers.image.title="IM Account Service"
LABEL org.opencontainers.image.description="Webitel IM Account Service"
LABEL org.opencontainers.image.vendor="Webitel"
LABEL org.opencontainers.image.source="https://github.com/webitel/im-account-service"

RUN --mount=type=cache,target=/var/cache/apk \
  apk --update add \
  ca-certificates \
  tzdata \
  && \
  update-ca-certificates

ARG UID=10001
RUN adduser \
  --disabled-password \
  --gecos "" \
  --home "/nonexistent" \
  --shell "/sbin/nologin" \
  --no-create-home \
  --uid "${UID}" \
  webitel
USER webitel

COPY --from=build /bin/im-account-service /bin/

ENTRYPOINT [ "/bin/im-account-service" ]