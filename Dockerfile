ARG BASE_IMAGE=docker:latest


FROM golang:1.17 AS builder

ARG VERSION=HEAD
ARG COMMIT=unknown

RUN apt-get update && apt-get install -y upx

RUN mkdir -p /output/usr/bin /output/usr/share /output/etc && cp -r /usr/share/zoneinfo /output/usr/share/zoneinfo && ln -s /usr/share/zoneinfo/UTC /output/etc/localtime
COPY ./assets/entrypoint.sh /output/entrypoint.sh

WORKDIR /usr/src/concron
COPY . .

RUN make CGO_ENABLED=0 VERSION=$VERSION COMMIT=$COMMIT DEFAULT_LISTEN=:80
RUN upx --lzma /usr/src/concron/concron && cp /usr/src/concron/concron /output/usr/bin



FROM $BASE_IMAGE

RUN for id in `seq 1000 1010`; do adduser -D -u $id -s /sbin/nologin -H -h / -g "" $id; done

ENV CONCRON_LISTEN=":80" CONCRON_PATH="/etc/crontab:/etc/cron.d"
EXPOSE 80
WORKDIR /tmp/concron
HEALTHCHECK CMD wget --spider http://localhost/livez || exit 1

COPY --from=builder /output /

ENTRYPOINT ["/entrypoint.sh"]
CMD ["/usr/bin/concron"]
