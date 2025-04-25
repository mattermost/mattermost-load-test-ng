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
      wget https://dl.grafana.com/oss/release/grafana_11.6.1_amd64.deb && \
      sudo dpkg -i grafana_11.6.1_amd64.deb && \
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
      wget https://github.com/grafana/alloy/releases/download/v1.3.1/alloy-1.3.1-1.amd64.deb && \
      sudo dpkg -i alloy-1.3.1-1.amd64.deb && \
      sudo systemctl enable alloy && \
      wget https://github.com/grafana/pyroscope/releases/download/v1.7.1/pyroscope_1.7.1_linux_amd64.deb && \
      sudo dpkg -i pyroscope_1.7.1_linux_amd64.deb && \
      sudo mkdir -p /var/lib/pyroscope && \
      sudo chown pyroscope:pyroscope -R /var/lib/pyroscope && \
      sudo systemctl enable pyroscope && \
      sudo mkdir /opt/yace && \
      wget https://github.com/nerdswords/yet-another-cloudwatch-exporter/releases/download/v0.61.2/yet-another-cloudwatch-exporter_0.61.2_Linux_x86_64.tar.gz && \
      sudo tar -zxf yet-another-cloudwatch-exporter_0.61.2_Linux_x86_64.tar.gz -C /opt/yace && \
      # Install Loki
      wget https://github.com/grafana/loki/releases/download/v3.2.0/loki_3.2.0_amd64.deb && \
      sudo dpkg -i loki_3.2.0_amd64.deb && \
      sudo systemctl start loki
      exit 0
   n=$((n+1))
   sleep 2
done

echo 'All retry attempts have failed, exiting' && exit 1
