ARG BASE_IMAGE=docker:latest


FROM golang:1.17 AS builder

ARG VERSION=HEAD
ARG COMMIT=unknown

RUN apt update && apt install -y upx

WORKDIR /usr/src/concron
COPY . .

RUN make CGO_ENABLED=0 VERSION=$VERSION COMMIT=$COMMIT DEFAULT_LISTEN=:80
RUN upx --lzma /usr/src/concron/concron


FROM $BASE_IMAGE

RUN for id in `seq 1000 1010`; do adduser -D -u $id -s /sbin/nologin -H -h / -g "" $id; done

ENV CONCRON_LISTEN=":80" CONCRON_PATH="/etc/crontab:/etc/cron.d"
EXPOSE 80
WORKDIR /tmp/concron
HEALTHCHECK CMD wget --spider http://localhost || exit 1

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /usr/src/concron/concron /usr/bin/concron
COPY ./assets/start-concron.sh /start-concron.sh

CMD ["/start-concron.sh"]
