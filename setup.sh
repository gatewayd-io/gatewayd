#!/bin/bash

# This script is used to install the required packages and download
# the latest version of GatewayD from GitHub and install the plugins.

# Set the architecture to amd64 if it is not set
if [ -z "${ARCH}" ]; then
    architecture=$(uname -m)
    if [[ "${architecture}" = "x86_64" ]]; then
        echo "Setting architecture to amd64"
        ARCH=amd64 && export ARCH
    elif [[ "${architecture}" = "aarch64" ]] || [[ "${architecture}" = "arm64" ]]; then
        echo "Setting architecture to arm64"
        ARCH=arm64 && export ARCH
    fi
fi
echo "Using architecture: ${ARCH}"

# Install the required packages
apk add --no-cache curl git

# Get the latest released version of GatewayD from GitHub
[ -z "${GATEWAYD_VERSION}" ] && GATEWAYD_VERSION=$(git ls-remote --tags --sort=v:refname "https://github.com/gatewayd-io/gatewayd" | cut -d/ -f3- | tail -n1) && export GATEWAYD_VERSION

# Check if the GatewayD version is set
if [ -z "${GATEWAYD_VERSION}" ]; then
    echo "Failed to set GatewayD version. Consider setting the GATEWAYD_VERSION environment variable manually."
    exit 126
fi

echo "Installing GatewayD ${GATEWAYD_VERSION}"

# Set the environment variables if they are not set
[ -z "${GATEWAYD_FILES}" ] && GATEWAYD_FILES=/gatewayd-files && export GATEWAYD_FILES

# Create the directory to store the gatewayd files
[ -d "${GATEWAYD_FILES}" ] || mkdir "${GATEWAYD_FILES}"

# Change the directory to the gatewayd-files directory to download the GatewayD archive
# and install the plugins. This will fail exit code 127 (file or directory not found).
cd "${GATEWAYD_FILES}" || exit 127

# Download the GatewayD archive if it doesn't exist
[ -f "${GATEWAYD_FILES}"/gatewayd-linux-"${ARCH}"-"${GATEWAYD_VERSION}".tar.gz ] || curl -L https://github.com/gatewayd-io/gatewayd/releases/download/"${GATEWAYD_VERSION}"/gatewayd-linux-"${ARCH}"-"${GATEWAYD_VERSION}".tar.gz -o gatewayd-linux-"${ARCH}"-"${GATEWAYD_VERSION}".tar.gz
if [ -f "${GATEWAYD_FILES}"/gatewayd-linux-"${ARCH}"-"${GATEWAYD_VERSION}".tar.gz ]; then
    echo "GatewayD archive downloaded successfully."
else
    echo "GatewayD archive download failed."
    exit 1
fi

# Extract the GatewayD archive
echo "Extracting GatewayD archive..."
tar zxvf gatewayd-linux-"${ARCH}"-"${GATEWAYD_VERSION}".tar.gz -C "${GATEWAYD_FILES}"

# Set execute permission for the GatewayD binary
echo "Setting execute permission for the GatewayD binary..."
chmod +x gatewayd

# Install the GatewayD plugins
# If the installation fails, the script will exit with exit code 126 (command invoke error).
echo "Installing GatewayD plugins..."
"${GATEWAYD_FILES}"/gatewayd plugin install --skip-path-slip-verification --output-dir "${GATEWAYD_FILES}" --plugin-config "${GATEWAYD_FILES}"/gatewayd_plugins.yaml --cleanup=false --update --overwrite-config || exit 126

# Replace the default Redis URL with the Redis container URL
[ -z "${REDIS_URL}" ] && REDIS_URL="redis://redis:6379/0" && export REDIS_URL
echo "Setting Redis URL to ${REDIS_URL}"
sed -i "s#redis://localhost:6379/0#${REDIS_URL}#" "${GATEWAYD_FILES}"/gatewayd_plugins.yaml

echo "GatewayD ${GATEWAYD_VERSION} and plugins installed successfully. Exiting..."
