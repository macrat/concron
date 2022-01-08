FROM ruby:latest

RUN curl -sLf https://raw.github.com/macrat/concron/main/assets/install.sh | sh -

COPY crontab /etc/crontab
COPY main.rb /main.rb

RUN ["/usr/local/bin/concron"]
