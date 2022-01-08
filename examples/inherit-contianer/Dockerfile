FROM ghcr.io/macrat/concron:slim

RUN apt-get update && apt-get install -y python3

COPY crontab /etc/crontab
COPY main.py /main.py
