#!/usr/bin/env bash

if [[ ! $EUID -eq 0 ]]; then
    echo "This script must be run as root"
    exit 1
fi

sudo apt install -y curl libvirt-dev


VERSION="${1:-latest}"
[[ "$VERSION" != v* && "$VERSION" != "latest" ]] && VERSION="v$VERSION"

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$|"latest" ]]; then
    echo "Invalid version format: $VERSION"
    exit 2
fi

cd /var/tmp

if [[ "$VERSION" == "latest" ]]; then
    VERSION=$(curl -sL https://api.github.com/repos/Ben-PP/libvirt-backup/releases/latest | grep -oP '"tag_name": "\K(.*)(?=")')
else
    # Check if the specified version exists
    if ! curl -sL https://api.github.com/repos/Ben-PP/libvirt-backup/releases/tags/$VERSION | grep -q '"tag_name": "'$VERSION'"'; then
        echo "Version $VERSION not found"
        exit 3
    fi
fi

# Download and set up the binary
sudo curl -Lo libvirt-backup https://github.com/Ben-PP/libvirt-backup/releases/download/$VERSION/libvirt-backup_linux-amd64
sudo mv libvirt-backup /usr/local/sbin/libvirt-backup
sudo chown root:root /usr/local/sbin/libvirt-backup
sudo chmod 700 /usr/local/sbin/libvirt-backup

# Download and set up the systemd service
sudo curl -Lo libvirt-backup.service https://raw.githubusercontent.com/Ben-PP/libvirt-backup/refs/tags/$VERSION/libvirt-backup.service
sudo mv libvirt-backup.service /etc/systemd/system/libvirt-backup.service
sudo chown root:root /etc/systemd/system/libvirt-backup.service
sudo chmod 644 /etc/systemd/system/libvirt-backup.service

# Create the default configuration file
sudo mkdir -p /etc/libvirt-backup
if [[ ! -f /etc/libvirt-backup/config.yaml ]]; then
    sudo curl -Lo config.yaml https://raw.githubusercontent.com/Ben-PP/libvirt-backup/refs/tags/$VERSION/config.example.yaml
    sudo mv config.yaml /etc/libvirt-backup/config.yaml
    sudo chown root:root /etc/libvirt-backup/config.yaml
    sudo chmod 644 /etc/libvirt-backup/config.yaml
fi

sudo systemctl daemon-reload