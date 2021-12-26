#!/bin/sh

if [ "${CONCRON_CRONTAB}" != "" ]; then
    printf "%s" "${CONCRON_CRONTAB}" > /tmp/concron/CONCRON_CRONTAB
    export CONCRON_PATH="${CONCRON_PATH}:CONCRON_CRONTAB"
fi

exec /usr/bin/concron
