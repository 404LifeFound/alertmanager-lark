FROM golang:1.25.5-bookworm AS builder

WORKDIR /src

COPY go.* ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 go build -v -o alertmanager-webhook

FROM debian:bookworm-slim
WORKDIR /app

# update ca-certificates
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

ARG user=alertmanager-webhook
ARG group=alertmanager-webhook
ARG uid=10000
ARG gid=10001

# If you bind mount a volume from the host or a data container,
# ensure you use the same uid
RUN groupadd -g ${gid} ${group} \
    && useradd -l -u ${uid} -g ${gid} -m -s /bin/bash ${user}

USER ${user}

COPY --from=builder --chown=${uid}:${gid} /src/alertmanager-webhook /app/alertmanager-webhook

ENV GIN_RELEASE_MODE=true

EXPOSE 8080

ENTRYPOINT ["/app/alertmanager-webhook","server"]
