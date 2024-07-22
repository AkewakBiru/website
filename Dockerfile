FROM golang:1.22

ARG APP_ENV

WORKDIR /app

COPY . .

RUN go build -o app && chown -R 65534:65534 /app && \
	mkdir -p /var/log/app && chown -R 65534:65534 /var/log/app

ENV APP_ENV=${APP_ENV} PWD="/app"

EXPOSE 3000

USER 65534

CMD [ "./app" ]