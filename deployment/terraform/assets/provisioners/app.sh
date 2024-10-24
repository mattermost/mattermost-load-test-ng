#!/bin/bash

set -euo pipefail

# Wait for boot to be finished (e.g. networking to be up).
while [ ! -f /var/lib/cloud/instance/boot-finished ]; do echo 'Waiting for cloud-init...'; sleep 1; done

system_arch=$(uname -m)
if [ "$arch" == "x86_64" ]; then
  arch="amd64"
fi
postgresql_version="14"
prometheus_node_exporter_version="1.8.2"
netpeek_version="0.1.4"

function install_prometheus_node_exporter() {
    echo "Installing Prometheus Node Exporter"
    wget https://github.com/prometheus/node_exporter/releases/download/v${prometheus_node_exporter_version}/node_exporter-${prometheus_node_exporter_version}.linux-${arch}.tar.gz && \
    tar xvfz node_exporter-${prometheus_node_exporter_version}.linux-${arch}.tar.gz && \
    sudo cp node_exporter-${prometheus_node_exporter_version}.linux-${arch}/node_exporter /usr/local/bin
}

function install_otel_collector() {
    echo "Installing OpenTelemetry Collector"
    wget https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/v${otel_collector_version}/otelcol-contrib_${otel_collector_version}_linux_${arch}.rpm && \
    sudo rpm -i otelcol-contrib_${otel_collector_version}_linux_${arch}.rpm && \
    sudo sed -i "s/User=.*/User=$(whoami)/g" /lib/systemd/system/otelcol-contrib.service && \
    sudo sed -i "s/Group=.*/Group=$(whoami)/g" /lib/systemd/system/otelcol-contrib.service && \
    sudo systemctl daemon-reload && \
    sudo systemctl restart otelcol-contrib
}

function install_netpeek() {
    wget https://github.com/streamer45/netpeek/releases/download/v${netpeek_version}/netpeek-v${netpeek_version} && \
    sudo mv netpeek-v* /usr/local/bin/netpeek && \
    sudo chmod +x /usr/local/bin/netpeek
}

# Retry loop (up to 3 times)
n=0
until [ "$n" -ge 3 ]
do
      # Note: commands below are expected to be either idempotent or generally safe to be run more than once.
      echo "Attempt ${n}"
      echo 'tcp_bbr' | sudo tee -a /etc/modules-load.d/tcp_bbr.conf && \
      sudo modprobe tcp_bbr && \
      sudo yum -y install https://download.postgresql.org/pub/repos/yum/reporpms/EL-9-x86_64/pgdg-redhat-repo-latest.noarch.rpm && \
      sudo yum -y install postgresql${postgresql_version}-server && \
      sudo /usr/pgsql-${postgresql_version}/bin/postgresql-${postgresql_version}-setup initdb && \
      sudo systemctl enable --now postgresql-${postgresql_version} && \
      sudo yum -y install wget && \
      sudo yum -y install postgresql14 && \
      sudo yum -y install numactl kernel-tools && \
      install_netpeek && \
      install_prometheus_node_exporter && \
      install_otel_collector && \
      exit 0
   n=$((n+1))
   sleep 2
done

echo 'All retry attempts have failed, exiting' && exit 1
