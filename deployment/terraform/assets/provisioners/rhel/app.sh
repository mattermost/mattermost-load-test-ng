#!/bin/bash

# Wait for boot to be finished (e.g. networking to be up).
while [ ! -f /var/lib/cloud/instance/boot-finished ]; do echo 'Waiting for cloud-init...'; sleep 1; done

# Load common
source common.sh

# Retry loop (up to 3 times)
n=0
until [ "$n" -ge 3 ]
do
      # Note: commands below are expected to be either idempotent or generally safe to be run more than once.
      echo "Attempt ${n}"
      echo 'tcp_bbr' | sudo tee -a /etc/modules-load.d/tcp_bbr.conf && \
      sudo modprobe tcp_bbr && \
      install_postgresql_server && \
      sudo dnf -y install wget && \
      sudo dnf -y install numactl kernel-tools && \
      install_netpeek && \
      install_prometheus_node_exporter && \
      install_otel_collector && \
      exit 0
   n=$((n+1))
   sleep 2
done

echo 'All retry attempts have failed, exiting' && exit 1
