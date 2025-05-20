#!/bin/bash

# Versions
export prometheus_node_exporter_version="1.8.2"
export otel_collector_version="0.110.0"
export netpeek_version="0.1.4"

export postgresql_version="14"
export keycloak_version="24.0.2"

export grafana_version="10.2.3"
export grafana_package="grafana"
export prometheus_version="1.8.2"
export prometheus_node_exporter_version="1.8.2"
export inbucket_version="2.1.0"
export elasticsearch_exporter_version="1.1.0"
export redis_exporter_version="1.58.0"
export alloy_version="1.3.1"
export alloy_rev="1"
export pyroscope_version="1.7.1"
export pyroscope_rev="1"
export yace_version="0.61.2"
export loki_version="3.2.0"
export keycloak_version="24.0.2"

export wget_common_args="--no-clobber"

# Calculated
# Detect system architecture first
system_arch="$(uname -m)"
if [ "$system_arch" == "x86_64" ]; then
  export arch="amd64"
fi
export system_arch

function update_system() {
    sudo dnf -y update
}

function install_otel_collector() {
    echo "Installing OpenTelemetry Collector"
    wget https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/v${otel_collector_version}/otelcol-contrib_${otel_collector_version}_linux_${arch}.rpm && \
    sudo rpm -i "otelcol-contrib_${otel_collector_version}_linux_${arch}.rpm" && \
    sudo sed -i "s/User=.*/User=$(whoami)/g" /lib/systemd/system/otelcol-contrib.service && \
    sudo sed -i "s/Group=.*/Group=$(whoami)/g" /lib/systemd/system/otelcol-contrib.service && \
    sudo systemctl daemon-reload && \
    sudo systemctl restart otelcol-contrib
}

function install_prometheus_node_exporter() {
    echo "Installing Prometheus Node Exporter"
    wget https://github.com/prometheus/node_exporter/releases/download/v${prometheus_node_exporter_version}/node_exporter-${prometheus_node_exporter_version}.linux-${arch}.tar.gz && \
    tar xvfz "node_exporter-${prometheus_node_exporter_version}.linux-${arch}.tar.gz" && \
    sudo cp "node_exporter-${prometheus_node_exporter_version}.linux-${arch}/node_exporter" /usr/local/bin
}

function install_netpeek() {
    wget https://github.com/streamer45/netpeek/releases/download/v${netpeek_version}/netpeek-v${netpeek_version} && \
    sudo mv netpeek-v* /usr/local/bin/netpeek && \
    sudo chmod +x /usr/local/bin/netpeek
}
