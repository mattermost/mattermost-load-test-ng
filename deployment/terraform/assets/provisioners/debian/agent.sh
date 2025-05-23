#!/bin/bash

set -euo pipefail

# Wait for boot to be finished (e.g. networking to be up).
while [ ! -f /var/lib/cloud/instance/boot-finished ]; do
	echo 'Waiting for cloud-init...'
	sleep 1
done

# Retry loop (up to 3 times)
n=0
until [ "$n" -ge 3 ]; do
	# Note: commands below are expected to be either idempotent or generally safe to be run more than once.
	echo "Attempt ${n}"
	sudo apt-get -y update &&
		sudo apt-get install -y prometheus-node-exporter &&
		sudo apt-get install -y numactl linux-tools-aws &&
		# Install OpenTelemetry collector, using ubuntu user to avoid permission issues
		wget https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/v0.120.0/otelcol-contrib_0.120.0_linux_amd64.deb &&
		sudo dpkg -i otelcol-contrib_0.120.0_linux_amd64.deb &&
		sudo sed -i 's/User=.*/User=ubuntu/g' /lib/systemd/system/otelcol-contrib.service &&
		sudo sed -i 's/Group=.*/Group=ubuntu/g' /lib/systemd/system/otelcol-contrib.service &&
		sudo systemctl daemon-reload && sudo systemctl restart otelcol-contrib &&
		exit 0
	n=$((n + 1))
	sleep 2
done

echo 'All retry attempts have failed, exiting' && exit 1
