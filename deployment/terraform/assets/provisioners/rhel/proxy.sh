#!/bin/bash

set -euo pipefail

# Wait for boot to be finished (e.g. networking to be up).
while [ ! -f /var/lib/cloud/instance/boot-finished ]; do echo 'Waiting for cloud-init...'; sleep 1; done

# Import common
source common.sh

# Retry loop (up to 3 times)
n=0
until [ "$n" -ge 3 ]
do
      # Note: commands below are expected to be either idempotent or generally safe to be run more than once.
      echo "Attempt ${n}"
      echo 'tcp_bbr' | sudo tee /etc/modules-load.d/tcp_bbr.conf && \
      sudo modprobe tcp_bbr && \
      sudo rpm --import https://nginx.org/keys/nginx_signing.key && \
      sudo sh -c 'echo "[nginx]
name=nginx
baseurl=https://nginx.org/packages/mainline/centos/\$releasever/\$basearch/
gpgcheck=1
enabled=1" > /etc/yum.repos.d/nginx.repo' && \
      sudo dnf -y update && \
      sudo dnf -y install wget && \
      sudo dnf -y install nginx && \
      sudo dnf -y install numactl kernel-tools && \
      install_prometheus_node_exporter && \
      install_otel_collector && \
      sudo systemctl daemon-reload && \
      sudo systemctl enable nginx && \
      sudo mkdir -p /etc/nginx/snippets && \
      sudo mkdir -p /etc/nginx/conf.d && \
      sudo mkdir -p /etc/nginx/sites-enabled && \
      sudo mkdir -p /etc/nginx/sites-available && \
      sudo rm -f /etc/nginx/conf.d/default.conf && \
      sudo ln -fs /etc/nginx/sites-available/mattermost.conf /etc/nginx/sites-enabled/mattermost.conf && \
      exit 0

   n=$((n+1))
   sleep 2
done

echo 'All retry attempts have failed, exiting' && exit 1
