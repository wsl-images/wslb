package app

import (
	"context"

	"github.com/wsl-images/wslb/internal/features"
	"github.com/wsl-images/wslb/internal/workspace"
)

type WorkspaceService interface {
	LoadAndValidate(ctx context.Context, manifestPath string) (*workspace.Manifest, string, workspace.ValidationResult, error)
	Plan(ctx context.Context, manifestPath, imageID string) (workspace.Plan, workspace.ValidationResult, error)
	Build(ctx context.Context, manifestPath, imageID, preferredEngine string) error
	Publish(ctx context.Context, manifestPath, imageID string) error
}

type WSLService interface {
	StateCreate(ctx context.Context, manifestPath, imageID string, fallbackWindowsDir bool, nonInteractive bool) (workspace.Image, error)
	Install(ctx context.Context, manifestPath, imageID, preferredEngine string, fallbackWindowsDir bool, nonInteractive bool) (string, error)
	Upgrade(ctx context.Context, manifestPath, imageID, preferredEngine string, fallbackWindowsDir bool, nonInteractive bool) (string, error)
	Rollback(ctx context.Context, manifestPath, imageID, backupPath string, fallbackWindowsDir bool, nonInteractive bool) error
	Status(ctx context.Context, manifestPath, imageID string) (string, error)
}

type DoctorService interface {
	Run(ctx context.Context) DoctorReport
}

type CatalogService interface {
	DescribeFeature(ctx context.Context, ref string) (features.Metadata, error)
}

type DoctorCheck struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	Remediation string `json:"remediation,omitempty"`
}

type DoctorReport struct {
	OK     bool          `json:"ok"`
	Checks []DoctorCheck `json:"checks"`
}
