#!/bin/bash

# Load common
source common.sh

# Retry loop (up to 3 times)
n=0
until [ "$n" -ge 3 ]
do
      # Note: commands below are expected to be either idempotent or generally safe to be run more than once.
      echo "Attempt ${n}"
      update_system && \
      install_postgresql_client && \
      sudo dnf -y install wget && \
      install_netpeek && \
      install_prometheus_node_exporter && \
      install_otel_collector && \
      exit 0
   n=$((n+1))
   sleep 2
done

echo 'All retry attempts have failed, exiting' && exit 1
