#!/bin/bash

# This script is used to install the required packages and download
# the latest version of GatewayD from GitHub and install the plugins.

# Get the latest released version of GatewayD from GitHub
[ -z "${GATEWAYD_VERSION}" ] && GATEWAYD_VERSION=$(git ls-remote --tags --sort=v:refname "https://github.com/gatewayd-io/gatewayd" | cut -d/ -f3- | tail -n1) && export GATEWAYD_VERSION

# Set the environment variables if they are not set
[ -z "${GATEWAYD_FILES}" ] && GATEWAYD_FILES=/gatewayd-files && export GATEWAYD_FILES

# Install the required packages
apk add --no-cache curl git

# Create the directory to store the gatewayd files
[ -d "${GATEWAYD_FILES}" ] || mkdir "${GATEWAYD_FILES}"

cd "${GATEWAYD_FILES}" || exit 1

# Download the GatewayD archive if it doesn't exist
[ -f "${GATEWAYD_FILES}"/gatewayd-linux-amd64-"${GATEWAYD_VERSION}".tar.gz ] || curl -L https://github.com/gatewayd-io/gatewayd/releases/download/"${GATEWAYD_VERSION}"/gatewayd-linux-amd64-"${GATEWAYD_VERSION}".tar.gz | tar zxvf -
chmod +x gatewayd

# Install the GatewayD plugins
"${GATEWAYD_FILES}"/gatewayd plugin install --skip-path-slip-verification --output-dir "${GATEWAYD_FILES}" --plugin-config "${GATEWAYD_FILES}"/gatewayd_plugins.yaml --cleanup=false --update --overwrite-config

# Replace the default Redis URL
sed -i 's/redis:\/\/localhost:6379/redis:\/\/redis:6379/' "${GATEWAYD_FILES}"/gatewayd_plugins.yaml
