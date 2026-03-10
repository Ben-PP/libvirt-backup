# libvirt-backup

Linux daemon to run scheduled backups of virtual machines using libvirt. Currently this only supports qemu/kvm.

# Requirements

- libvirt-dev

```bash
sudo apt install libvirt-dev
```

# Usage

This is meant to be run as a systemd service and configured via config yaml.

## Notes about behavior

There are couple behaviors which are expected but might come as a surprise.

### Only first backup of the day is saved when pruning

First backup of the day is named as `/backupdir/<vm-name>/dd-mm-yyyy_disk.qcow2` and any subsequent backups for the day are named as `/backupdir/<vm-name>/dd-mm-yyyy_disk_XX.qcow2` where `XX` is any number starting from 1. This leads to only the first backup of the day to be saved and all other backups of the day to be pruned. This behavior is expected and originates from pruning being done via constructing the file names which are to be saved. This might change later but for now this is the way.

## Installation

### Install script

Install using the [install.sh](./install.sh) script.

```bash
# Coming soon
```

### Manual

For now here is the vague installation instructions. Better will come when there is better installation method and publication pipeline.

#### Create config file

Config is read from `/etc/libvirt-backup/config.yaml` by default, but this can be changed with `-c` flag. Create the config file and use the [config.example.yaml](./config.example.yaml) as a base.

```bash
sudo mkdir /etc/libvirt-backup
sudo touch /etc/libvirt-backup/config.yaml
# Fill in the config
```

#### Decide the backup directory

In the config file `backup-dir` key defines the directory under which all of the backups will be stored. This can be any directory you like, but be mindful of the fact that the directory will be chowned to `libvirt-qemu:kvm` and if it does not exist, it will be created.

#### Add the binary

Add the binary to `/usr/local/sbin`.

Remember to allow execution.

```bash
sudo chmod 700 /usr/local/sbin/libvirt-backup
```

#### Create systemd service

This program is meant to be run as a systemd service. Create a service file for it. You can use the provided [libvirt-backup.service](./libvirt-backup.service).
