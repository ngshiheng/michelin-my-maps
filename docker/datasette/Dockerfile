FROM datasetteproject/datasette

RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

RUN pip install datasette-cluster-map \
    datasette-block-robots \
    datasette-google-analytics \
    datasette-dashboards

# Install s5cmd
RUN curl -fsSL https://github.com/peak/s5cmd/releases/download/v2.3.0/s5cmd_2.3.0_Linux-64bit.tar.gz \
    -o /tmp/s5cmd.tar.gz \
    && tar -xzf /tmp/s5cmd.tar.gz -C /tmp s5cmd \
    && install -m 755 /tmp/s5cmd /usr/local/bin/s5cmd \
    && rm /tmp/s5cmd.tar.gz /tmp/s5cmd

COPY docker/datasette/docker-entrypoint.sh /docker/datasette/docker-entrypoint.sh
COPY docker/datasette/metadata.json /docker/datasette/metadata.json
RUN chmod +x /docker/datasette/docker-entrypoint.sh

ENTRYPOINT ["/docker/datasette/docker-entrypoint.sh"]
EXPOSE 8001
