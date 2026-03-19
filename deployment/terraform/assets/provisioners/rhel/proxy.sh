#!/bin/bash

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
      uname -r && \
      sudo sh -c 'echo "[nginx]
baseurl=http://nginx.org/packages/centos/\$releasever/\$basearch/
gpgcheck=1
enabled=1
gpgkey=https://nginx.org/keys/nginx_signing.key
module_hotfixes=true" > /etc/yum.repos.d/nginx.repo' && \
      sudo dnf -y update && \
      sudo dnf -y install numactl kernel-tools wget nginx && \
      install_prometheus_node_exporter && \
      install_otel_collector && \
      sudo systemctl daemon-reload && \
      sudo systemctl enable nginx && \
      sudo mkdir -p /etc/nginx/snippets && \
      sudo mkdir -p /etc/nginx/conf.d && \
      sudo mkdir -p /etc/nginx/sites-enabled && \
      sudo mkdir -p /etc/nginx/sites-available && \
      sudo rm -f /etc/nginx/conf.d/default.conf && \
      sudo ln -fs /etc/nginx/sites-available/mattermost /etc/nginx/sites-enabled/mattermost && \
      exit 0
   n=$((n+1))
   sleep 2
done

echo 'All retry attempts have failed, exiting' && exit 1
