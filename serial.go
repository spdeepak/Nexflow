package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func main() {
	serial, err := getSerialNumber()
	if err != nil {
		fmt.Printf("Error retrieving serial number: %v\n", err)
		return
	}
	fmt.Printf("Your laptop serial number is: %s\n", serial)
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
