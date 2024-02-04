FROM golang:1.21-alpine AS go-builder
ENV CGO_ENABLED=1
RUN apk add -q --no-cache gcc musl-dev
WORKDIR $GOPATH/src/github.com/ngshiheng/michelin-my-maps
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags='-s -w -extldflags "-static"' -o /bin/mym ./cmd/mym

FROM datasetteproject/datasette
RUN datasette install datasette-publish-vercel
COPY --from=go-builder /bin/mym /bin/mym
ARG VERCEL_TOKEN
ENV VERCEL_TOKEN=${VERCEL_TOKEN}
RUN apt-get update -yq && apt-get -yq install curl gnupg ca-certificates && curl -L https://deb.nodesource.com/setup_18.x | bash && apt-get install -yq nodejs && npm i --silent -g vercel && apt clean && rm -rf /var/lib/apt && rm -rf /var/lib/dpkg/info/*

COPY ./docker/docker-entrypoint.sh .
CMD ["./docker-entrypoint.sh"]
