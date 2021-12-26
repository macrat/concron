ARG BASE_IMAGE=docker:latest


FROM golang:1.17 AS builder

WORKDIR /usr/src/concron
COPY . .

ARG VERSION=HEAD
ARG COMMIT=unknown

RUN CGO_ENABLED=0 make VERSION=$VERSION COMMIT=$COMMIT

RUN apt update && apt install -y upx && upx --lzma /usr/src/concron/concron


FROM $BASE_IMAGE

WORKDIR /tmp/concron

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /usr/src/concron/concron /usr/bin/concron
COPY ./start-concron.sh /start-concron.sh

RUN for id in `seq 1000 1010`; do adduser -D -u $id -s /sbin/nologin -H -h / -g "" $id; done

ENV CONCRON_LISTEN=":80" CONCRON_PATH="/etc/crontab:/etc/cron.d"
EXPOSE 80
HEALTHCHECK CMD wget --spider http://localhost || exit 1

CMD ["/start-concron.sh"]
