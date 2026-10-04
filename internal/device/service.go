package device

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/google/uuid"
)

type (
	service struct {
		querier Querier
		// serialFn resolves this machine's hardware serial. It is a field so
		// tests can exercise the fallback paths without shelling out.
		serialFn func(context.Context) (string, error)
	}
	// Service resolves the stable identifier for this machine.
	Service interface {
		// GetDeviceDetail returns the hardware serial number of this machine.
		// When the serial cannot be determined (permission denied, unsupported
		// OS, placeholder value from the firmware), a random identifier is
		// generated and persisted so the value stays stable across restarts.
		GetDeviceDetail(ctx context.Context) string
	}
)

func NewService(querier Querier) Service {
	return &service{
		querier:  querier,
		serialFn: getSerialNumber,
	}
}

func (s *service) GetDeviceDetail(ctx context.Context) string {
	id, err := s.querier.GetDeviceDetail(ctx)
	if err == nil {
		return id
	}

	serial, err := s.serialFn(ctx)
	if err != nil {
		// Not fatal: the identifier is persisted, so a generated fallback
		// still gives this install a stable device id on the next run.
		serial = ""
	}
	if !isUsableSerial(serial) {
		serial = uuid.NewString()
	}

	stored, err := s.querier.AddDevice(ctx, serial)
	if err != nil {
		return serial
	}
	return stored
}

// getSerialNumber returns the hardware serial number for the current OS.
func getSerialNumber(ctx context.Context) (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return serialNumberDarwin(ctx)
	case "windows":
		return serialNumberWindows(ctx)
	case "linux":
		return serialNumberLinux(ctx)
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// serialNumberDarwin reads the serial from system_profiler's JSON output,
// whose keys are stable regardless of the system language. The text output
// labels are localized, so grepping for "Serial Number" fails on non-English
// systems.
func serialNumberDarwin(ctx context.Context) (string, error) {
	out, err := runCommand(ctx, "system_profiler", "-json", "SPHardwareDataType")
	if err != nil {
		return "", err
	}

	var payload struct {
		Hardware []struct {
			SerialNumber string `json:"serial_number"`
		} `json:"SPHardwareDataType"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return "", fmt.Errorf("failed to parse system_profiler output: %w", err)
	}
	if len(payload.Hardware) == 0 {
		return "", fmt.Errorf("system_profiler returned no hardware data")
	}
	return strings.TrimSpace(payload.Hardware[0].SerialNumber), nil
}

// serialNumberWindows asks PowerShell for the BIOS serial. WMIC was the
// original command but is deprecated and removed by default on Windows 11.
func serialNumberWindows(ctx context.Context) (string, error) {
	out, err := runCommand(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command",
		"(Get-CimInstance Win32_BIOS).SerialNumber")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// serialNumberLinux reads the DMI product serial. On most distributions this
// file is root-readable only, so the call is expected to fail for a normal
// desktop user; the caller falls back to a generated identifier.
func serialNumberLinux(ctx context.Context) (string, error) {
	data, err := os.ReadFile("/sys/class/dmi/id/product_serial")
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("failed to read /sys/class/dmi/id/product_serial: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// runCommand executes name with args, cancelling on ctx and folding stderr
// into the error so failures are diagnosable instead of a bare "exit status 1".
func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("%s: %w", name, ctxErr)
		}
		if detail := strings.TrimSpace(stderr.String()); detail != "" {
			return nil, fmt.Errorf("%s: %w: %s", name, err, detail)
		}
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return stdout.Bytes(), nil
}

// isUsableSerial rejects empty and placeholder values reported by firmware.
// Placeholders such as "To be filled by O.E.M." are not unique per machine, so
// using one as the device id would collide across installations.
func isUsableSerial(serial string) bool {
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return false
	}
	switch strings.ToLower(serial) {
	case "none",
		"to be filled by o.e.m.",
		"default string",
		"system serial number",
		"base board serial number",
		"chassis serial number",
		"not specified",
		"not applicable",
		"oem",
		"invalid",
		"0":
		return false
	}
	return true
}
