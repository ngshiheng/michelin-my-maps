# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache \
    gcc \
    musl-dev \
    curl

WORKDIR /app

COPY go.mod go.sum ./
ENV CGO_ENABLED=1
RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=1 go build \
    -ldflags='-s -w -extldflags "-static"' \
    -trimpath \
    -o /bin/mym \
    ./cmd/mym

# Runtime stage
FROM alpine:3.22

RUN apk add --no-cache \
    sqlite \
    jq \
    curl

RUN adduser -D -s /bin/sh appuser

# Install mc
RUN curl -fsSL https://dl.min.io/client/mc/release/linux-amd64/mc \
    -o /usr/local/bin/mc \
    && chmod +x /usr/local/bin/mc

COPY --from=builder /bin/mym /bin/mym
COPY --from=builder /app/docker/scraper/docker-entrypoint.sh /docker/scraper/docker-entrypoint.sh

RUN chmod +x /docker/scraper/docker-entrypoint.sh

USER appuser

WORKDIR /app

CMD ["/docker/scraper/docker-entrypoint.sh"]
