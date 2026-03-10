package backups

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"time"

	"libvirt-backup/config"
)

func currentDate(offset int) string {
	date := time.Now()
	finalDate := date.AddDate(0, 0, offset)
	return finalDate.Format(time.DateOnly)
}

func filesToKeep(vm config.MachineConfig) []string {
	var filesToSave []string
	for _, disk := range vm.Disks {
		for i := 0; i < vm.Keep; i += 1 {
			fileName := fmt.Sprintf("%s_%s.qcow2", currentDate(-i), disk.Name)
			filesToSave = append(filesToSave, fileName)
		}
	}
	return filesToSave
}

func Prune(vm config.MachineConfig, vmDir string) error {
	filesToKeep := filesToKeep(vm)
	files, err := os.ReadDir(vmDir)
	if err != nil {
		return fmt.Errorf("Failed to read directory %s: %v", vmDir, err)
	}
	for _, file := range files {
		if slices.Contains(filesToKeep, file.Name()) || file.IsDir() {
			continue
		}
		fileToPrune := fmt.Sprintf("%s/%s", vmDir, file.Name())
		slog.Info("Removing file", "path", fileToPrune)
		if err := os.Remove(fileToPrune); err != nil {
			slog.Error("Failed to remove file", "path", fileToPrune)
			return fmt.Errorf("Failed to prune file %s: %v", fileToPrune, err)
		}
	}
	return nil
}

func newFileName(path string, diskName string) (string, error) {
	fileExt := ".qcow2"
	fileBase := fmt.Sprintf("%s/%s_%s", path, currentDate(0), diskName)
	counter := 0
	addition := ""
	finalName := fmt.Sprintf("%s%s%s", fileBase, addition, fileExt)
	for {
		if _, err := os.Stat(finalName); errors.Is(err, os.ErrNotExist) {
			break
		}
		if counter > 500 {
			return "", fmt.Errorf("Failed to create non existing file name: %s", finalName)
		}
		counter += 1
		addition = fmt.Sprintf("_%d", counter)
		finalName = fmt.Sprintf("%s%s%s", fileBase, addition, fileExt)
	}
	return finalName, nil
}

func NewXml(vm config.MachineConfig, vmDir string) (string, error) {
	var diksEntries string
	for _, disk := range vm.Disks {
		fileName, err := newFileName(vmDir, disk.Name)
		if err != nil {
			return "", fmt.Errorf("Failed to create backup xml: %v", err)
		}
		diksEntries += fmt.Sprintf(`
		<disk name="%s" type="%s">
			<target file="%s"/>
		</disk>`, disk.Name, disk.Type, fileName)
	}
	return fmt.Sprintf(`<domainbackup>
	<disks>%s
	</disks>
</domainbackup>`, diksEntries), nil
}
