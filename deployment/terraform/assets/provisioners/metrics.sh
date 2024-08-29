#!/bin/bash

set -euo pipefail

# Wait for boot to be finished (e.g. networking to be up).
while [ ! -f /var/lib/cloud/instance/boot-finished ]; do echo 'Waiting for cloud-init...'; sleep 1; done

# Retry loop (up to 3 times)
n=0
until [ "$n" -ge 3 ]
do
      # Note: commands below are expected to be either idempotent or generally safe to be run more than once.
      echo "Attempt ${n}"
      sudo apt-get -y update && \
      sudo apt-get install -y prometheus && \
      sudo systemctl enable prometheus && \
      sudo apt-get install -y adduser libfontconfig1 musl && \
      wget https://dl.grafana.com/oss/release/grafana_10.2.3_amd64.deb && \
      sudo dpkg -i grafana_10.2.3_amd64.deb && \
      wget https://github.com/inbucket/inbucket/releases/download/v2.1.0/inbucket_2.1.0_linux_amd64.deb && \
      sudo dpkg -i inbucket_2.1.0_linux_amd64.deb && \
      wget https://github.com/justwatchcom/elasticsearch_exporter/releases/download/v1.1.0/elasticsearch_exporter-1.1.0.linux-amd64.tar.gz && \
      sudo mkdir /opt/elasticsearch_exporter && \
      sudo tar -zxf elasticsearch_exporter-1.1.0.linux-amd64.tar.gz -C /opt/elasticsearch_exporter --strip-components=1 && \
      wget https://github.com/oliver006/redis_exporter/releases/download/v1.58.0/redis_exporter-v1.58.0.linux-amd64.tar.gz && \
      sudo mkdir /opt/redis_exporter && \
      sudo tar -zxf redis_exporter-v1.58.0.linux-amd64.tar.gz -C /opt/redis_exporter --strip-components=1 && \
      sudo systemctl daemon-reload && \
      sudo systemctl enable grafana-server && \
      sudo service grafana-server start && \
      sudo systemctl enable inbucket && \
      sudo service inbucket start && \
      wget https://dl.pyroscope.io/release/pyroscope_0.37.2_amd64.deb && \
      sudo apt-get install ./pyroscope_0.37.2_amd64.deb && \
      sudo systemctl enable pyroscope-server && \
      exit 0
   n=$((n+1)) 
   sleep 2
done

echo 'All retry attempts have failed, exiting' && exit 1
