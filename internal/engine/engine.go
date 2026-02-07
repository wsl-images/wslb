package engine

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type Name string

const (
	Docker Name = "docker"
	Podman Name = "podman"
)

type Status struct {
	Name        Name   `json:"name"`
	Installed   bool   `json:"installed"`
	Ready       bool   `json:"ready"`
	Version     string `json:"version,omitempty"`
	Message     string `json:"message,omitempty"`
	Remediation string `json:"remediation,omitempty"`
}

type Runner struct {
	name Name
}

func NewRunner(name Name) *Runner {
	return &Runner{name: name}
}

func Detect(ctx context.Context, preferred string) (Name, []Status, error) {
	statuses := []Status{checkEngine(ctx, Docker), checkEngine(ctx, Podman)}

	pick := func(n Name) bool {
		for _, st := range statuses {
			if st.Name == n && st.Installed && st.Ready {
				return true
			}
		}
		return false
	}

	switch strings.ToLower(preferred) {
	case "docker":
		if pick(Docker) {
			return Docker, statuses, nil
		}
	case "podman":
		if pick(Podman) {
			return Podman, statuses, nil
		}
	case "":
		if pick(Docker) {
			return Docker, statuses, nil
		}
		if pick(Podman) {
			return Podman, statuses, nil
		}
	default:
		return "", statuses, fmt.Errorf("unsupported engine: %s", preferred)
	}

	return "", statuses, fmt.Errorf("no ready container engine detected")
}

func checkEngine(ctx context.Context, n Name) Status {
	bin := string(n)
	if _, err := exec.LookPath(bin); err != nil {
		return Status{
			Name:        n,
			Installed:   false,
			Ready:       false,
			Message:     fmt.Sprintf("%s CLI not found", bin),
			Remediation: fmt.Sprintf("install %s and ensure it is on PATH", bin),
		}
	}
	verOut, _ := exec.CommandContext(ctx, bin, "--version").CombinedOutput()
	version := strings.TrimSpace(string(verOut))

	ready := false
	msg := ""
	rem := ""
	switch n {
	case Docker:
		if err := exec.CommandContext(ctx, bin, "info").Run(); err == nil {
			ready = true
		} else {
			msg = "docker daemon unavailable"
			rem = "start Docker Desktop or Docker service"
		}
	case Podman:
		if err := exec.CommandContext(ctx, bin, "info").Run(); err == nil {
			ready = true
		} else {
			msg = "podman service unavailable"
			rem = "initialize/start podman machine or local podman service"
		}
	}

	return Status{Name: n, Installed: true, Ready: ready, Version: version, Message: msg, Remediation: rem}
}

func (r *Runner) BuildImage(ctx context.Context, imageTag, dockerfilePath, contextDir string) error {
	cmd := exec.CommandContext(ctx, string(r.name), "build", "-t", imageTag, "-f", dockerfilePath, contextDir)
	cmd.Stdout = nil
	cmd.Stderr = nil
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s build failed: %w: %s", r.name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (r *Runner) CreateContainer(ctx context.Context, containerName, imageTag string) error {
	out, err := exec.CommandContext(ctx, string(r.name), "create", "--name", containerName, imageTag).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s create failed: %w: %s", r.name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (r *Runner) ExportContainer(ctx context.Context, containerName, outputTar string) error {
	out, err := exec.CommandContext(ctx, string(r.name), "export", "-o", outputTar, containerName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s export failed: %w: %s", r.name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (r *Runner) RemoveContainer(ctx context.Context, containerName string) {
	_ = exec.CommandContext(ctx, string(r.name), "rm", "-f", containerName).Run()
}

func (r *Runner) Pull(ctx context.Context, image string) error {
	out, err := exec.CommandContext(ctx, string(r.name), "pull", image).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s pull failed: %w: %s", r.name, err, strings.TrimSpace(string(out)))
	}
	return nil
}
