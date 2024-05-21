FROM alpine:3.19.1

RUN apk add --no-cache ca-certificates \
 && adduser -D -u 1000 jx

COPY ./web/static /app/web/static
COPY ./web/templates /app/web/templates
COPY ./build/linux/lighthouse-webui-plugin /app/

WORKDIR /app
USER 1000

ENTRYPOINT ["/app/lighthouse-webui-plugin"]