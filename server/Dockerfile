FROM caddy:2.8.4-builder-alpine AS builder

WORKDIR /app

RUN /usr/bin/xcaddy build --output caddy \
    --with github.com/corazawaf/coraza-caddy/v2@latest \
    --with github.com/mholt/caddy-ratelimit

FROM caddy:2.8.4-alpine

COPY --from=builder /app/caddy /usr/bin/caddy
COPY @owasp_crs .
COPY @crs-setup.conf.example .
COPY akewak.biru.etco.cloud/ akewak.biru.etco.cloud
COPY caddyfile .

ENV UPSTREAMS=test

EXPOSE 80
EXPOSE 443
EXPOSE 443/udp
EXPOSE 2019

CMD ["caddy", "run", "--config", "./caddyfile", "--adapter", "caddyfile"]