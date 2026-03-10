package config

import (
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/go-co-op/gocron/v2"
	"gopkg.in/yaml.v3"
	"libvirt.org/go/libvirt"
)

type DiskConfig struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

type MachineConfig struct {
	Name     string       `yaml:"name"`
	Schedule string       `yaml:"schedule"`
	Keep     int          `yaml:"keep"`
	Disks    []DiskConfig `yaml:"disks"`
}

type Config struct {
	BackupDir string                   `yaml:"backup-dir"`
	Machines  map[string]MachineConfig `yaml:"machines"`
}

func (c Config) Validate() error {
	// TODO Could add a check for vms existing in libvirt, but not in config, and warn about them
	if c.BackupDir == "" {
		return fmt.Errorf("invalid backup directory")
	}
	if len(c.Machines) == 0 {
		return fmt.Errorf("at least one machine configuration is required")
	}
	for mk, m := range c.Machines {
		c := gocron.NewDefaultCron(false)
		if err := c.IsValid(m.Schedule, time.Now().Location(), time.Now()); err != nil {
			return fmt.Errorf("invalid cron syntax in '%s'", mk)
		}
		if m.Name == "" {
			return fmt.Errorf("machine name is required for '%s'", mk)
		}
		if m.Keep < 1 {
			return fmt.Errorf("keep is invalid for '%s'", mk)
		}
		if len(m.Disks) == 0 {
			return fmt.Errorf("at least one disk is required for '%s'", mk)
		}
		for _, disk := range m.Disks {
			if disk.Name == "" {
				return fmt.Errorf("disk name is required for '%s'", mk)
			}
			if !slices.Contains([]string{"file"}, disk.Type) {
				return fmt.Errorf("disk type is invalid in disk '%s' of '%s'", disk.Name, mk)
			}
		}
	}
	return nil
}

func New(configPath string) (*Config, error) {
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("Config error: %v", err)
	}

	config := Config{}
	if err := yaml.Unmarshal(configBytes, &config); err != nil {
		return nil, fmt.Errorf("Config error: %v", err)
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("Config validation error: %v", err)
	}

	virtConn, err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to libvirt: %v", err)
	}
	doms, err := virtConn.ListAllDomains(0)
	if err != nil {
		return nil, fmt.Errorf("Failed to list domains: %v", err)
	}
	virtConn.Close()
	domNames := make([]string, len(doms))
	for i, dom := range doms {
		name, err := dom.GetName()
		if err != nil {
			return nil, fmt.Errorf("Failed to get name for domain %d: %v", i, err)
		}
		domNames[i] = name
	}
	for _, vm := range config.Machines {
		if !slices.Contains(domNames, vm.Name) {
			return nil, fmt.Errorf("Configured machine '%s' does not exist in libvirt", vm.Name)
		}
	}
	return &config, nil
}
