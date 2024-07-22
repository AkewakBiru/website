FROM caddy:2.8.4-builder-alpine AS builder

ARG UPSTREAMS

WORKDIR /app

RUN /usr/bin/xcaddy build --output caddy \
    --with github.com/corazawaf/coraza-caddy/v2@latest \
    --with github.com/mholt/caddy-ratelimit

FROM caddy:2.8.4-alpine

COPY --from=builder /app/caddy /usr/bin/caddy

ARG UPSTREAMS

COPY @owasp_crs .
COPY @crs-setup.conf.example .
COPY akewak.biru.etco.cloud/ akewak.biru.etco.cloud
COPY caddyfile .

RUN mkdir -p /var/log/caddy && \
    chown -R 65534:65534 ./akewak.biru.etco.cloud /var/log/caddy/ /data/caddy /config/caddy

ENV UPSTREAMS=${UPSTREAMS}

EXPOSE 80
EXPOSE 443
EXPOSE 443/udp
EXPOSE 2019

USER 65534

CMD ["caddy", "run", "--config", "./caddyfile", "--adapter", "caddyfile"]