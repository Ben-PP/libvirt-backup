package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"os/user"
	"syscall"

	"libvirt-backup/backups"
	"libvirt-backup/config"
	"libvirt-backup/files"

	"github.com/go-co-op/gocron/v2"
	"libvirt.org/go/libvirt"
)

const (
	VERSION = "1.0.0"
	USAGE   = `Usage:
	libvirt-backup [options]

	Options:
		-c, --config <path>   Path to config file (default: /etc/libvirt-backup/config.yaml)
		-h, --help            Show usage
		-v, --version         Show version and exit
		--validate            Validate config and exit
`
)

func startBackup(vm config.MachineConfig, backupDir string) error {
	slog.Info("Starting backup", "vm", vm.Name)
	virtConn, err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		return fmt.Errorf("Failed to connect to libvirt: %v", err)
	}
	defer virtConn.Close()
	domain, err := virtConn.LookupDomainByName(vm.Name)
	if err != nil {
		slog.Warn("Domain not found. Skipping backup.", "vm", vm.Name)
		return nil
	}
	defer domain.Free()

	vmDir := fmt.Sprintf("%s/%s", backupDir, vm.Name)
	if err := files.Mkdir(vmDir); err != nil {
		return fmt.Errorf("Failed to create backup directory %s: %v", vmDir, err)
	}
	if err := files.ChownToLibvirt(vmDir); err != nil {
		return fmt.Errorf("Failed to chown directory %s to libvirt: %v", vmDir, err)
	}

	// Prune old files
	if err := backups.Prune(vm, vmDir); err != nil {
		slog.Warn("Failed to prune backups. Continuing without pruning.", "vm", vm.Name, "error", err)
	}

	backupXml, err := backups.NewXml(vm, vmDir)
	if err != nil {
		return fmt.Errorf("Failed to get backup xml for vm %s: %v", vm.Name, err)
	}

	// Throws error if config defines non existing disks. Whole machine is skipped
	if err := domain.BackupBegin(backupXml, "", 0); err != nil {
		return fmt.Errorf("Failed to start backup for vm %s: %v", vm.Name, err)
	}
	slog.Info("Backup process started", "vm", vm.Name)

	return nil
}

func checkRoot() error {
	curUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("Failed to get current user: %v", err)
	}
	if curUser.Username != "root" {
		return fmt.Errorf("This program must be run as root")
	}
	return nil
}

func initFlags(configPath *string, showHelp *bool, validate *bool, showVersion *bool) {
	flag.StringVar(configPath, "config", "/etc/libvirt-backup/config.yaml", "Path to config file")
	flag.StringVar(configPath, "c", "/etc/libvirt-backup/config.yaml", "Path to config file")
	flag.BoolVar(showHelp, "help", false, "Show usage")
	flag.BoolVar(showHelp, "h", false, "Show usage")
	flag.BoolVar(showVersion, "version", false, "Show version and exit")
	flag.BoolVar(showVersion, "v", false, "Show version and exit")
	flag.BoolVar(validate, "validate", false, "Validate config and exit")
	flag.Parse()
	flag.Usage = func() { fmt.Fprintf(os.Stderr, USAGE) }
}

func main() {
	var configPath string
	var showHelp bool
	var validate bool
	var showVersion bool

	initFlags(&configPath, &showHelp, &validate, &showVersion)

	if showHelp {
		flag.Usage()
		return
	}
	if showVersion {
		fmt.Printf("libvirt-backup v%s\n", VERSION)
		return
	}

	removeTime := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey && len(groups) == 0 {
			return slog.Attr{}
		}
		return a
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: removeTime,
	}))
	slog.SetDefault(logger)

	if err := checkRoot(); err != nil {
		slog.Error(fmt.Sprintf("Permission error: %v", err))
		os.Exit(2)
	}

	config, err := config.New(configPath)
	if err != nil {
		slog.Error("Failed to read config", "path", configPath, "error", err)
		os.Exit(2)
	}
	slog.Info("Configuration is valid!")
	if validate {
		return
	}

	slog.Info("Starting backup scheduler")

	if err := files.Mkdir(config.BackupDir); err != nil {
		slog.Error(fmt.Sprintf("Failed to initialize backup directory: %s", err))
		os.Exit(2)
	}
	if err := files.ChownToLibvirt(config.BackupDir); err != nil {
		slog.Error(fmt.Sprintf("Failed to chown backup directory: %s", err))
		os.Exit(2)
	}

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		slog.Error("Failed to create scheduler", "error", err)
		os.Exit(2)
	}

	schedules_created := 0
	schedules_failed := 0
	for _, machine := range config.Machines {
		_, err := scheduler.NewJob(
			gocron.CronJob(machine.Schedule, false),
			gocron.NewTask(func() {
				if err := startBackup(machine, config.BackupDir); err != nil {
					slog.Error("Failed to start backup", "vm", machine.Name, "error", err)
				}
			}),
		)
		if err != nil {
			slog.Error("Failed to schedule backup", "vm", machine.Name)
			schedules_failed += 1
			continue
		}
		schedules_created += 1
		slog.Info("Schedule created", "vm", machine.Name)
	}
	scheduler.Start()
	slog.Info("Started backup scheduler", "schedules_created", schedules_created, "schedules_failed", schedules_failed)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs)
	for sig := range sigs {
		switch sig {
		case syscall.SIGURG:
			// Do nothing
		case syscall.SIGTERM:
			slog.Info("Exiting gracefully")
			os.Exit(0)
		default:
			slog.Warn("Exiting from signal", "signal", sig)
			os.Exit(1)
		}
	}
}
