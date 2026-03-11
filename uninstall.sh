#!/usr/bin/env bash

if [[ ! $EUID -eq 0 ]]; then
    echo "This script must be run as root"
    exit 1
fi

sudo rm -f /usr/local/sbin/libvirt-backup
sudo rm -f /etc/systemd/system/libvirt-backup.service
sudo rm -rf /etc/libvirt-backup
sudo systemctl daemon-reload