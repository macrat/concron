ARG BASE_IMAGE=docker:latest


FROM golang:1.17 AS builder

ARG VERSION=HEAD
ARG COMMIT=unknown

RUN apt-get update && apt-get install -y upx

RUN mkdir -p /output/usr/bin /output/usr/share /output/etc && cp -r /usr/share/zoneinfo /output/usr/share/zoneinfo && ln -s /usr/share/zoneinfo/UTC /output/etc/localtime

WORKDIR /usr/src/concron
COPY . .

RUN make CGO_ENABLED=0 VERSION=$VERSION COMMIT=$COMMIT DEFAULT_LISTEN=:80
RUN upx --lzma /usr/src/concron/concron && cp /usr/src/concron/concron /output/usr/bin



FROM $BASE_IMAGE

RUN for id in `seq 1000 1010`; do \
        echo "${id}:x:${id}:${id}::/:`which nologin`" >> /etc/passwd; \
        echo "${id}:x:${id}:${id}" >> /etc/group; \
        echo "${id}:*:::::::" >> /etc/shadow; \
    done

EXPOSE 80
HEALTHCHECK CMD ["/usr/bin/concron", "-health-check"]

COPY --from=builder /output /

CMD ["/usr/bin/concron"]
