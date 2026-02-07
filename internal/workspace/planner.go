package workspace

import (
	"fmt"
	"path/filepath"
	"time"
)

func BuildPlan(m *Manifest, manifestPath string, onlyImageID string) Plan {
	p := Plan{
		WorkspaceName: m.Workspace.Name,
		GeneratedAt:   time.Now().UTC(),
		Images:        make([]ImagePlan, 0),
	}

	outDir := ResolvePath(manifestPath, m.Workspace.OutputDir)
	for _, img := range m.Images {
		if onlyImageID != "" && img.ID != onlyImageID {
			continue
		}
		ip := ImagePlan{
			ImageID:          img.ID,
			Target:           img.Target,
			DisplayName:      img.DisplayName,
			Prerequisites:    []string{},
			Steps:            []string{},
			DeterministicDir: filepath.Join(outDir, "state", img.ID),
		}
		switch img.Target {
		case "wsl":
			stable := ""
			if img.WSL != nil {
				stable = img.WSL.DistroName
			}
			ip.Prerequisites = append(ip.Prerequisites,
				"wsl available",
				"container engine available (docker/podman)",
			)
			ip.Steps = append(ip.Steps,
				"build rootfs image from base",
				"resolve and apply features in deterministic order",
				"export container filesystem",
				"write .wsl artifact",
				"install or upgrade distribution with safety gates",
			)
			ip.ArtifactPath = filepath.Join(outDir, img.ID+".wsl")
			if stable != "" {
				ip.CandidateName = stable + "__candidate"
				ip.BackupPattern = stable + "__backup__<yyyyMMddHHmmss>"
			}
		default:
			ip.Prerequisites = append(ip.Prerequisites, "valid target type")
			ip.Steps = append(ip.Steps, "fix manifest target")
		}

		p.Images = append(p.Images, ip)
	}
	return p
}

func BackupName(stable string, t time.Time) string {
	return fmt.Sprintf("%s__backup__%s", stable, t.UTC().Format("20060102150405"))
}

func CandidateName(stable string) string {
	return stable + "__candidate"
}
