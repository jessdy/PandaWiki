FROM --platform=$BUILDPLATFORM golang:1.24.3-alpine AS builder

WORKDIR /src
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG TARGETOS TARGETARCH VERSION
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags "-s -w -extldflags '-static' -X github.com/chaitin/panda-wiki/telemetry.Version=${VERSION}" -o /build/panda-wiki-api pro/cmd/api_pro/main.go pro/cmd/api_pro/wire_gen.go \
    && GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags "-s -w -extldflags '-static' -X github.com/chaitin/panda-wiki/telemetry.Version=${VERSION}" -o /build/panda-wiki-migrate cmd/migrate/main.go cmd/migrate/wire_gen.go

FROM openeuler/openeuler:24.03-lts AS api

RUN dnf -y update \
    && dnf -y install ca-certificates tzdata shadow-utils \
    && update-ca-trust force-enable || true \
    && groupadd -g 10001 panda \
    && useradd -u 10001 -g panda -M -d /home/panda -s /sbin/nologin panda \
    && mkdir -p /home/panda /app/data/backups \
    && chown -R panda:panda /home/panda /app/data \
    && dnf clean all

# runtime 中 pg.go 使用编译时源码路径 /src/store/pg/migration，需在镜像中保留该路径
RUN mkdir -p /src/store/pg/migration

WORKDIR /app

COPY --from=builder --chown=panda:panda /build/panda-wiki-api /app/panda-wiki-api
COPY --from=builder --chown=panda:panda /build/panda-wiki-migrate /app/panda-wiki-migrate
COPY --from=builder --chown=panda:panda /src/store/pg/migration /app/migration
COPY --from=builder --chown=panda:panda /src/store/pg/migration /src/store/pg/migration

USER panda

CMD ["sh", "-c", "echo '🚀 panda-wiki-api (pro) container starting' && echo '📦 Running DB migration...' && /app/panda-wiki-migrate && echo '✅ Migration done' && echo '🎯 Starting API server (port 8000)...' && echo '' && exec /app/panda-wiki-api"]
