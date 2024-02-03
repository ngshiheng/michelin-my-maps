FROM golang:1.21-alpine AS go-builder
WORKDIR $GOPATH/src/github.com/ngshiheng/michelin-my-maps
COPY . .
RUN go build -o /bin/mym ./cmd/mym

FROM datasetteproject/datasette
RUN datasette install datasette-publish-vercel

COPY --from=go-builder /bin/mym /bin/mym

ARG VERCEL_TOKEN
ENV VERCEL_TOKEN=${VERCEL_TOKEN}
RUN apt-get update -yq
RUN apt-get -yq install curl gnupg ca-certificates
RUN curl -L https://deb.nodesource.com/setup_18.x | bash
RUN apt-get install -yq nodejs
RUN npm i -g vercel

COPY . .
CMD ["./docker/docker-entrypoint.sh"]
