#!/bin/sh
#
# This script is used to install the required packages and download
# the latest version of GatewayD from GitHub and install the plugins.
# If you run ./setup.sh -d or ./setup.sh --docker, it prepare Docker compose related files and configs.
# Written in a POSIX-compatible way (sh, dash, bash, etc.).

set -eu  #exit on any error and treat unset variables as errors


usage() {
    cat <<EOF
Usage: $0 [OPTIONS]

If no option is provided, the script installs GatewayD by default.

OPTIONS:
  -d, --docker   Download docker compose config files
  -h, --help     Show this help message
EOF
}

install_gatewayd() {
# Set the architecture to amd64 if it is not set
    if [ -z "${ARCH:-}" ]; then
        architecture="$(uname -m)"
        if [ "$architecture" = "x86_64" ]; then
            echo "Setting architecture to amd64"
            ARCH=amd64
            export ARCH
        elif [ "$architecture" = "aarch64" ] || [ "$architecture" = "arm64" ]; then
            echo "Setting architecture to arm64"
            ARCH=arm64
            export ARCH
        fi
    fi
    echo "Using architecture: ${ARCH}"

    # Install the required packages
    apk add --no-cache curl git

    # Get the latest released version of GatewayD from GitHub
    if [ -z "${GATEWAYD_VERSION:-}" ]; then
        GATEWAYD_VERSION="$(
            git ls-remote --tags --sort=v:refname "https://github.com/gatewayd-io/gatewayd" \
            | cut -d/ -f3- \
            | tail -n1
        )"
        export GATEWAYD_VERSION
    fi
    # Check if the GatewayD version is set
    if [ -z "${GATEWAYD_VERSION:-}" ]; then
        echo "Failed to set GatewayD version. Set GATEWAYD_VERSION manually if needed."
        exit 126
    fi

    echo "Installing GatewayD ${GATEWAYD_VERSION}"

    # Change the directory to the gatewayd-files directory to download the GatewayD archive
    # and install the plugins. This will fail exit code 127 (file or directory not found).

    if [ -z "${GATEWAYD_FILES:-}" ]; then
        GATEWAYD_FILES="/gatewayd-files"
        export GATEWAYD_FILES
    fi
    [ -d "$GATEWAYD_FILES" ] || mkdir -p "$GATEWAYD_FILES"

    # Move into that directory
    cd "$GATEWAYD_FILES" || exit 127

    # Download the GatewayD archive if it doesn't exist
    tarball="gatewayd-linux-${ARCH}-${GATEWAYD_VERSION}.tar.gz"
    if [ ! -f "$tarball" ]; then
        curl -L "https://github.com/gatewayd-io/gatewayd/releases/download/${GATEWAYD_VERSION}/${tarball}" \
            -o "$tarball"
    fi

    if [ -f "$tarball" ]; then
        echo "GatewayD archive downloaded successfully."
    else
        echo "GatewayD archive download failed."
        exit 1
    fi

    #Extract and set permissions
    echo "Extracting GatewayD archive..."
    tar zxvf gatewayd-linux-"${ARCH}"-"${GATEWAYD_VERSION}".tar.gz -C "${GATEWAYD_FILES}"
    # Set execute permission for the GatewayD binary
    echo "Setting execute permission for the GatewayD binary..."
    chmod +x gatewayd
    
    # Install the GatewayD plugins
    # If the installation fails, the script will exit with exit code 126 (command invoke error).
    echo "Installing GatewayD plugins..."
    "${GATEWAYD_FILES}/gatewayd" plugin install \
        --skip-path-slip-verification \
        --output-dir "${GATEWAYD_FILES}" \
        --plugin-config "${GATEWAYD_FILES}/gatewayd_plugins.yaml" \
        --cleanup=false \
        --update \
        --overwrite-config || exit 126

    # Replace the default Redis URL with the Redis container URL
    if [ -z "${REDIS_URL:-}" ]; then
        REDIS_URL="redis://redis:6379/0"
        export REDIS_URL
    fi
    echo "Setting Redis URL to ${REDIS_URL}"
    sed -i "s#redis://localhost:6379/0#${REDIS_URL}#" "${GATEWAYD_FILES}/gatewayd_plugins.yaml"

    echo "GatewayD ${GATEWAYD_VERSION} and plugins installed successfully!"
}


# Docker compose config download

docker_mode() {
        curl -L https://raw.githubusercontent.com/gatewayd-io/gatewayd/main/docker-compose.yaml -o docker-compose.yaml
        mkdir -p observability-configs && cd observability-configs
        curl -L https://raw.githubusercontent.com/gatewayd-io/gatewayd/main/observability-configs/grafana-datasources.yaml -o datasources.yaml
        curl -L https://raw.githubusercontent.com/gatewayd-io/gatewayd/main/observability-configs/prometheus.yaml -o prometheus.yaml
        curl -L https://raw.githubusercontent.com/gatewayd-io/gatewayd/main/observability-configs/tempo.yaml -o tempo.yaml
}


# Main logic: Parse arguments

if [ $# -eq 0 ]; then
    # No arguments: Run the installation by default
    install_gatewayd
    exit 0
fi

# If there are arguments, parse them
while [ $# -gt 0 ]; do
    case "$1" in
        -d|--docker)
            docker_mode
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done
