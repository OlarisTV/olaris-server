#!/bin/bash
echo "Updating olaris-server from nightlies"
cd /opt/olaris
curl -vL --fail 'https://gitlab.com/olaris/olaris-server/-/jobs/artifacts/develop/download?job=dist-linux-amd64' > dist-linux-amd64.zip && \
  mkdir dist-linux-amd64 && \
  unzip -o dist-linux-amd64.zip -d dist-linux-amd64 && \
  unzip -o dist-linux-amd64/olaris-linux-amd64*.zip -d dist-linux-amd64 && \
  mv dist-linux-amd64/bin/olaris /opt/olaris/olaris-from-ci
exec "$@"
