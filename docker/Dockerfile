FROM golang:1.24-alpine AS go-builder
ENV CGO_ENABLED=1
RUN apk add -q --no-cache gcc musl-dev
WORKDIR $GOPATH/src/github.com/ngshiheng/michelin-my-maps
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags='-s -w -extldflags "-static"' -o /bin/mym ./cmd/mym

FROM python:3.13-alpine
RUN apk update && apk add --no-cache gcc musl-dev libffi-dev curl sqlite jq
RUN pip install datasette
COPY --from=go-builder /bin/mym /bin/mym
ARG GITHUB_TOKEN
ENV GITHUB_TOKEN=${GITHUB_TOKEN}
COPY ./docker/docker-entrypoint.sh .
COPY ./docker/metadata.json .
CMD ["sh","/docker-entrypoint.sh"]
