FROM datasetteproject/datasette

RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

# Install mc (Minio client)
RUN curl -fsSL https://dl.min.io/client/mc/release/linux-amd64/mc \
    -o /usr/local/bin/mc \
    && chmod +x /usr/local/bin/mc

COPY docker/datasette/docker-entrypoint.sh /docker/datasette/docker-entrypoint.sh
RUN chmod +x /docker/datasette/docker-entrypoint.sh

ENTRYPOINT ["/docker/datasette/docker-entrypoint.sh"]
EXPOSE 8001
