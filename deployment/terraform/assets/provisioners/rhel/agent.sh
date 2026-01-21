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
        # Foundation build tool which also include 'make'
        sudo dnf -y groupinstall "Development Tools" && \
        sudo dnf -y install numactl kernel-tools wget curl && \
        echo "Installing nvm Node.js version manager" && \
        curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash && \
        export NVM_DIR="$HOME/.nvm" && \
        [ -s "$NVM_DIR/nvm.sh" ] && source "$NVM_DIR/nvm.sh" && \
        echo "nvm installed successfully with version $(nvm --version)" && \
        # Although we have a .nvmrc file, but we cannot use that because its not available at the provisioner level
        nvm install 24.11 && \
        nvm use 24.11 && \
        echo "Node.js installed successfully with version $(node --version)" && \
        install_prometheus_node_exporter && \
        install_otel_collector && \
        exit 0
   n=$((n+1))
   sleep 2
done

echo 'All retry attempts have failed, exiting' && exit 1
