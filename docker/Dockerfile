FROM golang:1.24-alpine

RUN apk add --no-cache \
    gcc \
    musl-dev \
    curl \
    sqlite \
    jq \
    && rm -rf /var/cache/apk/*

RUN adduser -D -s /bin/sh appuser

WORKDIR /app

RUN curl -fsSL https://dl.min.io/client/mc/release/linux-amd64/mc \
    -o /usr/local/bin/mc \
    && chmod +x /usr/local/bin/mc

COPY go.mod go.sum ./

ENV CGO_ENABLED=1
RUN go mod download && go mod verify

COPY . .

RUN chmod +x ./docker/docker-entrypoint.sh

RUN CGO_ENABLED=1 go build \
    -ldflags='-s -w -extldflags "-static"' \
    -trimpath \
    -o /bin/mym \
    ./cmd/mym

ARG GITHUB_TOKEN

USER appuser

CMD ["./docker/docker-entrypoint.sh"]