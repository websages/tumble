# syntax=docker/dockerfile:1

# ---- Build stage: compile the tumble binary ----
FROM golang:1.25-alpine AS build
WORKDIR /src

# git lets the build stamp the commit hash into the binary
RUN apk add --no-cache git

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

# Build the static binary (pure-Go sqlite, so CGO stays disabled)
COPY . .
ARG GIT_COMMIT=unknown
RUN CGO_ENABLED=0 go build \
      -ldflags "-X tumble/internal/version.CommitHash=${GIT_COMMIT}" \
      -o /tumble ./cmd/tumble

# ---- Litestream stage: fetch the prebuilt binary ----
FROM alpine:3.20 AS litestream
ARG LITESTREAM_VERSION=0.5.14
RUN apk add --no-cache curl && \
    curl -fsSL -o /tmp/litestream.tar.gz \
      "https://github.com/benbjohnson/litestream/releases/download/v${LITESTREAM_VERSION}/litestream-${LITESTREAM_VERSION}-linux-x86_64.tar.gz" && \
    tar -C /usr/local/bin -xzf /tmp/litestream.tar.gz litestream && \
    rm /tmp/litestream.tar.gz

# ---- Runtime stage ----
FROM alpine:3.20

# ca-certificates: outbound HTTPS (link previews, S3/Tigris).
# tzdata: scheduler runs jobs in America/Chicago.
RUN apk add --no-cache ca-certificates tzdata

COPY --from=build /tumble /usr/local/bin/tumble
COPY --from=litestream /usr/local/bin/litestream /usr/local/bin/litestream
COPY conf/litestream.yml /etc/litestream.yml
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

EXPOSE 8080

# The entrypoint restores from the backup if the volume is empty, then runs
# tumble under Litestream for continuous replication. tumble is configured
# entirely via TUMBLE_* environment variables.
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
