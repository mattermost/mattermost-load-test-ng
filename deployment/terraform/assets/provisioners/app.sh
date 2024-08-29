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
      echo 'tcp_bbr' | sudo tee -a /etc/modules && \
      sudo modprobe tcp_bbr && \
      wget -qO - https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor | sudo tee /usr/share/keyrings/postgres-archive-keyring.gpg && \
      sudo sh -c 'echo "deb [signed-by=/usr/share/keyrings/postgres-archive-keyring.gpg] http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list' && \
      sudo apt-get -y update && \
      sudo apt-get install -y mysql-client-8.0 && \
      sudo apt-get install -y postgresql-client-14 && \
      sudo apt-get install -y prometheus-node-exporter && \
      sudo apt-get install -y numactl linux-tools-aws linux-tools-aws-lts-22.04 && \
      exit 0
   n=$((n+1)) 
   sleep 2
done

echo 'All retry attempts have failed, exiting' && exit 1
