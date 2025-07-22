#!/bin/bash

# Load common
source common.sh

# Retry loop (up to 3 times)
n=0
until [ "$n" -ge 3 ]
do
        # Note: commands below are expected to be either idempotent or generally safe to be run more than once.
        echo "Attempt ${n}"
        sudo dnf -y update && \
        sudo dnf -y install numactl kernel-tools wget curl && \
        # Install nvm - Node.js version manager
        curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash && \
        export NVM_DIR="$HOME/.nvm" && \
        [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh" && \
        # Install the Node.js version defined in .nvmrc and use it
        nvm install && \
        nvm use && \
        install_prometheus_node_exporter && \
        install_otel_collector && \
        exit 0
   n=$((n+1))
   sleep 2
done

echo 'All retry attempts have failed, exiting' && exit 1
