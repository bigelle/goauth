# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.27-alpine AS build
WORKDIR /src

# gcc + musl-dev needed for CGO (github.com/mattn/go-sqlite3)
RUN --mount=type=cache,target=/var/cache/apk \
    apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# CGO_ENABLED=1 is needed for github.com/mattn/go-sqlite3 (cgo driver).
# If your ent schema actually uses modernc.org/sqlite (pure Go driver) —
# check with: grep sqlite go.sum — then set CGO_ENABLED=0 and drop gcc/musl-dev above.
ENV CGO_ENABLED=1
ENV GOOS=linux

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# ---- runtime stage ----
FROM alpine:3.20 AS runtime

# ca-certificates: outbound TLS calls (e.g. redis over TLS)
# netcat-openbsd: used by the docker-compose healthcheck (nc -z localhost 50051)
# sqlite3 CLI is intentionally NOT installed here — install manually via
# `apk add sqlite` when shelling in for debugging.
RUN --mount=type=cache,target=/var/cache/apk \
    apk add --no-cache ca-certificates netcat-openbsd

RUN addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=build /out/server ./server

# sqlite db file — mount this as a volume so the e2e test runner can read it
# from outside the container. Adjust the path to whatever internal/config
# actually points the ent client at.
RUN mkdir -p /app/data && chown -R app:app /app

USER app
VOLUME ["/app/data"]
EXPOSE 50051
ENTRYPOINT ["./server"]
