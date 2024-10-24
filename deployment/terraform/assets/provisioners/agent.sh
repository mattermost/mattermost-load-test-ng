#!/bin/bash

set -euo pipefail

# Wait for boot to be finished (e.g. networking to be up).
while [ ! -f /var/lib/cloud/instance/boot-finished ]; do echo 'Waiting for cloud-init...'; sleep 1; done

system_arch=$(uname -m)
if [ "$arch" == "x86_64" ]; then
  arch="amd64"
fi
prometheus_node_exporter_version="1.8.2"
otel_collector_version="0.110.0"

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

# Retry loop (up to 3 times)
n=0
until [ "$n" -ge 3 ]
do
        # Note: commands below are expected to be either idempotent or generally safe to be run more than once.
        echo "Attempt ${n}"
        sudo dnf -y update && \
        sudo dnf -y install numactl kernel-tools && \
        sudo dnf -y install wget && \
        install_prometheus_node_exporter && \
        install_otel_collector && \
        exit 0
   n=$((n+1))
   sleep 2
done

echo 'All retry attempts have failed, exiting' && exit 1
