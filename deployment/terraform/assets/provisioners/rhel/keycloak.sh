#!/bin/bash

set -euo pipefail

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
      sudo dnf -y update && \
      sudo dnf -y install unzip java-17-openjdk postgresql postgresql-server wget && \
      sudo /usr/bin/postgresql-setup --initdb && \
      sudo systemctl enable --now postgresql && \
      sudo mkdir -p /opt/keycloak && \
      sudo curl -O -L --output-dir /opt/keycloak https://github.com/keycloak/keycloak/releases/download/${keycloak_version}/keycloak-${keycloak_version}.zip && \
      sudo unzip /opt/keycloak/keycloak-${keycloak_version}.zip -d /opt/keycloak && \
      sudo mkdir -p /opt/keycloak/keycloak-${keycloak_version}/data/import && \
      sudo chown -R "$(whoami):$(whoami)" /opt/keycloak && \
      install_prometheus_node_exporter && \
      exit 0
   n=$((n+1))
   sleep 2
done

echo 'All retry attempts have failed, exiting' && exit 1
