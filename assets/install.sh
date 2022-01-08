#!/bin/sh

set -e

latest_version() {
    curl -sLf -H 'Accept: application/json' https://github.com/macrat/concron/releases/latest | sed -e 's/.*"tag_name":"v\([-+.a-zA-Z0-9]*\)".*/\1/'
}

version=${CONCRON_VERSION:-$(latest_version)}

download_url="https://github.com/macrat/concron/releases/download/v${version}/concron_${version}_$(uname -s | tr '[:upper:]' '[:lower:]')_$(uname -m)"


echo "Start install Concron version ${version}..."
echo ""
echo "OS: $(uname -s)"
echo "Arch: $(uname -m)"
echo "URL: ${download_url}"
echo ""

curl -SLf ${download_url} -o /usr/local/bin/concron
chmod 755 /usr/local/bin/concron

echo ""
echo "Installation success!"
