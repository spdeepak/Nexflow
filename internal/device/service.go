//nolint:all
package device

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/google/uuid"
)

type (
	service struct {
		querier Querier
	}
	Service interface {
		GetDeviceDetail(ctx context.Context) uuid.UUID
	}
)

func NewService(querier Querier) Service {
	return &service{
		querier: querier,
	}
}

func (s *service) GetDeviceDetail(ctx context.Context) uuid.UUID {
	id, err := s.querier.GetDeviceDetail(ctx)
	if err == nil {
		return id
	}
	id, _ = s.querier.AddDevice(ctx, uuid.New())
	return id
}

func getSerialNumber() (string, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Uses WMIC to fetch the BIOS serial number
		cmd = exec.Command("wmic", "bios", "get", "serialnumber")
	case "darwin":
		// Uses system_profiler to fetch the hardware serial number on macOS
		cmd = exec.Command("sh", "-c", "system_profiler SPHardwareDataType | grep 'Serial Number'")
	case "linux":
		// Reads from sysfs (requires root/sudo permissions on most distributions)
		cmd = exec.Command("cat", "/sys/class/dmi/id/product_serial")
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	fmt.Print(out.String())
	return cleanOutput(out.String(), runtime.GOOS), nil
}

func cleanOutput(output string, osType string) string {
	lines := strings.Split(output, "\n")

	switch osType {
	case "windows":
		// WMIC returns a header line "SerialNumber" followed by the actual serial
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.EqualFold(line, "SerialNumber") {
				return line
			}
		}
	case "darwin":
		// macOS returns "Serial Number (system): XXXXXXXX"
		if len(lines) > 0 {
			parts := strings.Split(lines[0], ":")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
	case "linux":
		// Linux directly returns the string, just needs trimming
		return strings.TrimSpace(output)
	}

	return strings.TrimSpace(output)
}
