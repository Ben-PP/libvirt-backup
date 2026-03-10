package files

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
)

func libvirtUidGid() (int, int, error) {
	libvirtUser, err := user.Lookup("libvirt-qemu")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to lookup libvirt-qemu user: %v", err)
	}
	kvmGroup, err := user.LookupGroup("kvm")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to lookup kvm group: %v", err)
	}
	uid, _ := strconv.Atoi(libvirtUser.Uid)
	gid, _ := strconv.Atoi(kvmGroup.Gid)
	return uid, gid, nil
}

func ChownToLibvirt(path string) error {
	uid, gid, err := libvirtUidGid()
	if err != nil {
		return fmt.Errorf("failed to set ownership of backup directory: %v", err)
	}
	if err := os.Chown(path, uid, gid); err != nil {
		return fmt.Errorf("failed to set ownership of backup directory: %v", err)
	}
	return nil
}

func Mkdir(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("failed to create backup directory: %v", err)
	}
	return nil
}
