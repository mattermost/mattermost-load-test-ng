#!/bin/bash

# Import common
source common.sh

function install_deps() {
    sudo dnf -y install wget fontconfig
}

function install_grafana {
    echo "Installing Grafana"
    wget ${wget_common_args} -O grafana-gpg.key https://rpm.grafana.com/gpg.key
    sudo rpm --import grafana-gpg.key
    sudo sh -c 'echo "[grafana]
name=grafana
baseurl=https://rpm.grafana.com
repo_gpgcheck=1
enabled=1
gpgcheck=1
gpgkey=https://rpm.grafana.com/gpg.key
sslverify=1
sslcacert=/etc/pki/tls/certs/ca-bundle.crt" > /etc/yum.repos.d/grafana.repo' && \
    sudo dnf -y install "grafana-${grafana_version}" && \
    sudo systemctl enable --now grafana-server
}

function install_prometheus() {
    echo "Installing Prometheus"
    if ! id "prometheus"; then
        sudo adduser --no-create-home --shell /bin/false prometheus;
    fi;
    sudo mkdir -p /etc/prometheus /var/lib/prometheus && \
    sudo chown prometheus:prometheus /etc/prometheus && \
    sudo chown prometheus:prometheus /var/lib/prometheus && \
    wget ${wget_common_args} "https://github.com/prometheus/prometheus/releases/download/v${prometheus_version}/prometheus-${prometheus_version}.linux-${arch}.tar.gz" && \
    tar -xzf "prometheus-${prometheus_version}.linux-${arch}.tar.gz" && \
    sudo cp "prometheus-${prometheus_version}.linux-${arch}/prometheus" /usr/local/bin/ && \
    sudo chown prometheus:prometheus /usr/local/bin/prometheus && \
    sudo cp "prometheus-${prometheus_version}.linux-${arch}/promtool" /usr/local/bin/ && \
    sudo chown prometheus:prometheus /usr/local/bin/promtool && \
    sudo cp -r "prometheus-${prometheus_version}.linux-${arch}/consoles" /etc/prometheus && \
    sudo cp -r "prometheus-${prometheus_version}.linux-${arch}/console_libraries" /etc/prometheus && \
    sudo chown -R prometheus:prometheus /etc/prometheus/consoles && \
    sudo chown -R prometheus:prometheus /etc/prometheus/console_libraries && \
    sudo sh -c 'echo "[Unit]
Description=Prometheus
Wants=network-online.target
After=network-online.target
[Service]
User=prometheus
Group=prometheus
Type=simple
ExecStart=/usr/local/bin/prometheus \
--config.file /etc/prometheus/prometheus.yml \
--storage.local.path /var/lib/prometheus/ \
--web.console.templates=/etc/prometheus/consoles \
--web.console.libraries=/etc/prometheus/console_libraries
[Install]
WantedBy=multi-user.target" > /etc/systemd/system/prometheus.service' && \
    sudo systemctl daemon-reload
}

function install_inbucket() {
    echo "Installing Inbucket"
    wget ${wget_common_args} "https://github.com/inbucket/inbucket/releases/download/v${inbucket_version}/inbucket_${inbucket_version}_linux_${arch}.rpm" && \
    sudo dnf localinstall -y "inbucket_${inbucket_version}_linux_${arch}.rpm" && \
    sudo systemctl start --now inbucket
}

function install_elasticsearch_exporter() {
    echo "Installing Elasticsearch Exporter"
    wget ${wget_common_args} "https://github.com/justwatchcom/elasticsearch_exporter/releases/download/v${elasticsearch_exporter_version}/elasticsearch_exporter-${elasticsearch_exporter_version}.linux-${arch}.tar.gz" && \
    sudo mkdir -p /opt/elasticsearch_exporter && \
    sudo tar -zxf "elasticsearch_exporter-${elasticsearch_exporter_version}.linux-${arch}.tar.gz" -C /opt/elasticsearch_exporter --strip-components=1
}

function install_redis_exporter() {
    echo "Installing Redis Exporter"
    wget ${wget_common_args} "https://github.com/oliver006/redis_exporter/releases/download/v${redis_exporter_version}/redis_exporter-v${redis_exporter_version}.linux-${arch}.tar.gz" && \
    sudo mkdir -p /opt/redis_exporter && \
    sudo tar -zxf "redis_exporter-v${redis_exporter_version}.linux-${arch}.tar.gz" -C /opt/redis_exporter --strip-components=1
}

function install_alloy() {
    echo "Installing Alloy"
    wget ${wget_common_args} "https://github.com/grafana/alloy/releases/download/v${alloy_version}/alloy-${alloy_version}-${alloy_rev}.${arch}.rpm" && \
    sudo dnf localinstall -y "alloy-${alloy_version}-${alloy_rev}.${arch}.rpm" && \
    sudo systemctl enable --now alloy
}

function install_pyroscope() {
    echo "Installing Pyroscope"
    wget "https://github.com/grafana/pyroscope/releases/download/v${pyroscope_version}/pyroscope_${pyroscope_version}_linux_${arch}.rpm" && \
    sudo dnf localinstall -y "pyroscope_${pyroscope_version}_linux_${arch}.rpm" && \
    sudo mkdir -p /var/lib/pyroscope && \
    sudo chown pyroscope:pyroscope -R /var/lib/pyroscope && \
    sudo systemctl enable --now pyroscope
}

function install_yace() {
    echo "Installing Yace"
    sudo mkdir -p /opt/yace && \
    wget "https://github.com/nerdswords/yet-another-cloudwatch-exporter/releases/download/v${yace_version}/yet-another-cloudwatch-exporter_${yace_version}_Linux_${system_arch}.tar.gz" && \
    sudo tar -zxf "yet-another-cloudwatch-exporter_${yace_version}_Linux_${system_arch}.tar.gz" -C /opt/yace
}

function install_loki() {
    echo "Installing Loki"
    wget "https://github.com/grafana/loki/releases/download/v${loki_version}/loki-${loki_version}.${system_arch}.rpm" && \
    sudo rpm -i "loki-${loki_version}.${system_arch}.rpm" && \
    sudo systemctl start loki
}

# Retry loop (up to 3 times)
n=0
until [ "$n" -ge 3 ]
do
      # Note: commands below are expected to be either idempotent or generally safe to be run more than once.
      echo "Attempt ${n}"
      update_system && \
      install_deps && \
      install_grafana && \
      install_prometheus && \
      install_prometheus_node_exporter && \
      install_inbucket && \
      install_elasticsearch_exporter && \
      install_redis_exporter && \
      install_alloy && \
      install_pyroscope && \
      install_yace && \
      install_loki && \
      exit 0
   n=$((n+1))
   sleep 2
done

echo 'All retry attempts have failed, exiting' && exit 1
