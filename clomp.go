package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

func getField(text, field string) (string, bool) {
	if strings.HasPrefix(text, field+":") {
		value := strings.TrimSpace(strings.TrimPrefix(text, field+":"))
		return value, true
	}
	return "", false
}

type DeviceInfo struct {
	Agent string
	Name  string
	Type  string
}

func ParseRominfoOutput(data string) ([]DeviceInfo, error) {
	deviceInfos := []DeviceInfo{}
	sp := strings.Split(data, "*******")
	agent := ""
	for _, s := range sp {
		s := strings.TrimSpace(s)
		if strings.HasPrefix(s, "Agent") {
			agent = strings.TrimPrefix(s, "Agent")
			continue
		}
		scanner := bufio.NewScanner(strings.NewReader(s))
		name := ""
		deviceType := ""

		for scanner.Scan() {
			text := strings.TrimPrefix(scanner.Text(), "  ")

			if value, ok := getField(text, "Name"); ok {
				name = value
			}
			if value, ok := getField(text, "Device Type"); ok {
				deviceType = value
			}

		}

		if agent == "" {
			continue
		}
		di := DeviceInfo{
			Agent: agent,
			Name:  name,
			Type:  deviceType,
		}
		deviceInfos = append(deviceInfos, di)
	}
	return deviceInfos, nil
}

func RunCommandStdout(args []string) (string, error) {
	cmd := exec.Command(args[0], args[1:]...)
	stdout := &strings.Builder{}
	cmd.Stdout = stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("run command: %w", err)
	}

	return stdout.String(), nil
}

func RunCommandTransparent(args []string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func DetectGPUArch() (string, error) {
	stdout, err := RunCommandStdout([]string{"rocminfo"})
	if err != nil {
		return "", fmt.Errorf("running rocminfo: %w", bufio.ErrAdvanceTooFar)
	}

	deviceInfos, err := ParseRominfoOutput(stdout)
	if err != nil {
		return "", fmt.Errorf("parse rocminfo output: %w", err)
	}

	for _, di := range deviceInfos {
		if di.Type == "GPU" {
			return di.Name, nil
		}
	}
	return "", fmt.Errorf("GPU not found")
}

func run() error {
	gpuArch, err := DetectGPUArch()
	if err != nil {
		return fmt.Errorf("detect GPU architecture: %w", err)
	}

	log.Printf("clomp: targeting %s\n", gpuArch)

	args := []string{
		"amdclang",
		"-fopenmp",
		"--offload-arch=" + gpuArch,
	}
	args = append(args, os.Args[1:]...)
	if err := RunCommandTransparent(args); err != nil {
		return fmt.Errorf("run command: %w", err)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
