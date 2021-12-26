Concron - Cron for Container
============================

- :heavy_check_mark: Dashboard included.
- :heavy_check_mark: Prometheus/OpenMetrics exporter included.


## Quickstart

### via Docker command

Place your crontab file to `./crontab`, and run the below command.

``` shell
$ docker run -v $(pwd)/crontab:/etc/crontab:ro -p 8080:80 ghcr.io/macrat/concron:latest
```

### via Docker Compose

You can make `docker-compose.yml` like below,

``` yaml
version: "3"

services:
  concron:
    image: ghcr.io/macrat/concron:latest
    environment:
      CRON_TZ: Asia/Tokyo
      CONCRON_CRONTAB: |
        45 */3 * * *  *  docker run --rm busybox echo "you can do your task here!"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./cron-files:/etc/cron.d:ro
    ports:
      - "8000:80"
    restart: always
```

And then, start via `docker-compose up -d` command.

### without container

You can just start `crontab` command.

``` shell
$ export CRONTAB_PATH=/etc/crontab:/etc/cron.d CRONTAB_LISTEN=":8000"  # These are default values.

$ crontab
```


## Crontab

Concron search crontab files from `/etc/crontab` or under the `/etc/cron.d`.
In the container, you can also use `CONCRON_CRONTAB` environment variable instead of file.

``` crontab
SHELL=/bin/sh
CRON_TZ=Asia/Tokyo

#   schedule      user        command
# |------------| |----| |-----------------|
   */10 * * * *   root   echo your-command

   @daily         *      echo another command
```

This file format is the same as common crontab with user name column.
But there is single difference; you can use `*` as username that means execute on the same user as the execution user of Concron.

The container has users that have the same name as UID, 1000 to 1010.
So you can use specify UID in crontab like `1000` as user name.


## Dashboard

You can see dashbord on <http://localhost:8000> in default.

![dashboard example](./assets/dashboard.jpg)
