#!/bin/bash

set -euo pipefail

# Wait for boot to be finished (e.g. networking to be up).
while [ ! -f /var/lib/cloud/instance/boot-finished ]; do echo 'Waiting for cloud-init...'; sleep 1; done

system_arch=$(uname -m)
if [ "$arch" == "x86_64" ]; then
  arch="amd64"
fi
prometheus_node_exporter_version="1.8.2"


# Retry loop (up to 3 times)
n=0
until [ "$n" -ge 3 ]
do
        # Note: commands below are expected to be either idempotent or generally safe to be run more than once.
        echo "Attempt ${n}"
        sudo yum -y update && \
        sudo yum -y install numactl kernel-tools && \
        sudo yum -y install wget && \
        wget https://github.com/prometheus/node_exporter/releases/download/v${prometheus_node_exporter_version}/node_exporter-${prometheus_node_exporter_version}.linux-${arch}.tar.gz && \
        tar xvfz node_exporter-${prometheus_node_exporter_version}.linux-${arch}.tar.gz && \
        sudo cp node_exporter-${prometheus_node_exporter_version}.linux-${arch}/node_exporter /usr/local/bin && \
        exit 0
   n=$((n+1))
   sleep 2
done

echo 'All retry attempts have failed, exiting' && exit 1
