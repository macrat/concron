FROM ruby:latest

RUN curl -sLf https://raw.github.com/macrat/concron/main/assets/install.sh | sh -
EXPOSE 8000

COPY crontab /etc/crontab
COPY main.rb /main.rb

CMD ["/usr/local/bin/concron"]
