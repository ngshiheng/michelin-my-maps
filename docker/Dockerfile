FROM golang:1.24-alpine
ENV CGO_ENABLED=1
RUN apk add -q --no-cache gcc musl-dev curl sqlite jq
WORKDIR $GOPATH/src/github.com/ngshiheng/michelin-my-maps
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags='-s -w -extldflags "-static"' -o /bin/mym ./cmd/mym
ARG GITHUB_TOKEN
ENV GITHUB_TOKEN=${GITHUB_TOKEN}
CMD ["sh","./docker/docker-entrypoint.sh"]
