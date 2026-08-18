# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.26.5-bookworm AS build
WORKDIR /src

# Cache module downloads separately from source changes
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=1 is needed for github.com/mattn/go-sqlite3 (cgo driver).
#
# If your ent schema actually uses modernc.org/sqlite (pure Go driver) —
# check with: grep sqlite go.sum — then:
#   1. set CGO_ENABLED=0 below
#   2. delete the `apt-get install -y gcc ...` line, it becomes dead weight
ENV CGO_ENABLED=1
RUN apt-get update && apt-get install -y --no-install-recommends gcc libc6-dev \
    && rm -rf /var/lib/apt/lists/*

# adjust the build path if your entrypoint isn't cmd/server
RUN go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# ---- runtime stage ----
FROM debian:bookworm-slim AS runtime

# ca-certificates: needed if the service makes outbound TLS calls (e.g. to redis over TLS)
# netcat-openbsd: used only for the docker-compose healthcheck, drop if unused
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates netcat-openbsd \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=build /out/server ./server

# sqlite db file — mount this as a volume so the e2e test runner can read it
# from outside the container. Adjust the path to whatever internal/config
# actually points the ent client at.
RUN mkdir -p /app/data
VOLUME ["/app/data"]

EXPOSE 50051
ENTRYPOINT ["./server"]
