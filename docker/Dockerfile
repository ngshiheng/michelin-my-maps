FROM golang:1.22-alpine AS go-builder
ENV CGO_ENABLED=1
RUN apk add -q --no-cache gcc musl-dev
WORKDIR $GOPATH/src/github.com/ngshiheng/michelin-my-maps
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags='-s -w -extldflags "-static"' -o /bin/mym ./cmd/mym

FROM node:18-alpine AS node-builder
RUN npm i --silent -g vercel

FROM python:3.12-alpine
RUN apk update && apk add --no-cache gcc musl-dev libffi-dev
RUN pip install datasette
RUN datasette install datasette-publish-vercel
COPY --from=go-builder /bin/mym /bin/mym
COPY --from=node-builder /usr/local/bin/node /usr/local/bin/node
COPY --from=node-builder /usr/local/lib/node_modules /usr/local/lib/node_modules
RUN ln -s /usr/local/lib/node_modules/vercel/dist/index.js /usr/local/bin/vercel
ARG VERCEL_TOKEN
ENV VERCEL_TOKEN=${VERCEL_TOKEN}
ARG GITHUB_TOKEN
ENV GITHUB_TOKEN=${GITHUB_TOKEN}
COPY ./docker/docker-entrypoint.sh .
COPY ./docker/metadata.json .
CMD ["sh","/docker-entrypoint.sh"]
