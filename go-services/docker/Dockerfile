# Golang
FROM golang:1.24-bookworm AS base

ARG SERVICE_NAME
WORKDIR /opt/service/

COPY . .
RUN go build -o ./bin/service ./applications/${SERVICE_NAME}/cmd/

# Runtime
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y curl && \
    curl -SL https://github.com/docker/compose/releases/download/v2.24.7/docker-compose-linux-x86_64 -o /usr/local/bin/docker-compose && \
    chmod +x /usr/local/bin/docker-compose && \
    apt-get purge -y curl && rm -rf /var/lib/apt/lists/*

COPY --from=base /opt/service/bin/service /bin/service

ENTRYPOINT ["/bin/service"]
