package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/wsl-images/wslb/internal/output"
	"github.com/wsl-images/wslb/internal/workspace"
)

var (
	wslWorkspaceFile      string
	wslEngine             string
	wslFallbackWindowsDir bool
	wslNonInteractive     bool
	wslRollbackTo         string
)

var wslManagedCmd = &cobra.Command{
	Use:   "wsl",
	Short: "Managed WSL state/install/upgrade/rollback operations",
}

var wslStateCmd = &cobra.Command{
	Use:   "state",
	Short: "State disk and persistence operations",
}

var wslStateCreateCmd = &cobra.Command{
	Use:   "create <imageId>",
	Short: "Create managed state storage for a WSL image",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		started := time.Now().UTC()
		imageID := args[0]
		emitter().EmitEvent(output.Event{TS: time.Now().UTC(), Level: "info", Command: "wsl state create", ImageID: imageID, Event: "start"})
		img, err := getServices().WSL.StateCreate(context.Background(), wslWorkspaceFile, imageID, wslFallbackWindowsDir, wslNonInteractive)
		if err != nil {
			emitter().EmitEvent(output.Event{TS: time.Now().UTC(), Level: "error", Command: "wsl state create", ImageID: imageID, Event: "failed", Message: err.Error()})
			emitSimpleError("wsl state create", imageID, started, err)
			return err
		}
		res := output.NewResult("wsl state create", imageID, started, []output.Step{{
			ID:     "state.create",
			Name:   "create or validate external state",
			Status: "completed",
		}}, []output.Artifact{{
			Name: "state",
			Path: img.WSL.State.Path,
		}}, nil)
		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(struct {
				output.Result
				State interface{} `json:"state"`
			}{
				Result: res,
				State:  img.WSL.State,
			})
		}
		emitter().EmitResult(res)
		emitter().EmitEvent(output.Event{TS: time.Now().UTC(), Level: "info", Command: "wsl state create", ImageID: imageID, Event: "completed"})
		return nil
	},
}

var wslInstallCmd = &cobra.Command{
	Use:   "install <imageId>",
	Short: "Install WSL distro for a workspace image",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		started := time.Now().UTC()
		imageID := args[0]
		emitter().EmitEvent(output.Event{TS: time.Now().UTC(), Level: "info", Command: "wsl install", ImageID: imageID, Event: "start"})
		artifact, err := getServices().WSL.Install(context.Background(), wslWorkspaceFile, imageID, wslEngine, wslFallbackWindowsDir, wslNonInteractive)
		if err != nil {
			emitter().EmitEvent(output.Event{TS: time.Now().UTC(), Level: "error", Command: "wsl install", ImageID: imageID, Event: "failed", Message: err.Error()})
			emitSimpleError("wsl install", imageID, started, err)
			return err
		}
		res := output.NewResult("wsl install", imageID, started, []output.Step{{
			ID:     "install",
			Name:   "import and configure distro",
			Status: "completed",
		}}, []output.Artifact{{Name: "artifact", Path: artifact}}, nil)
		emitter().EmitResult(res)
		emitter().EmitEvent(output.Event{TS: time.Now().UTC(), Level: "info", Command: "wsl install", ImageID: imageID, Event: "completed"})
		return nil
	},
}

var wslUpgradeCmd = &cobra.Command{
	Use:   "upgrade <imageId>",
	Short: "Upgrade managed WSL distro with blue/green safety gates",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		started := time.Now().UTC()
		imageID := args[0]
		emitter().EmitEvent(output.Event{TS: time.Now().UTC(), Level: "info", Command: "wsl upgrade", ImageID: imageID, Event: "start"})
		backupPath, err := getServices().WSL.Upgrade(context.Background(), wslWorkspaceFile, imageID, wslEngine, wslFallbackWindowsDir, wslNonInteractive)
		if err != nil {
			emitter().EmitEvent(output.Event{TS: time.Now().UTC(), Level: "error", Command: "wsl upgrade", ImageID: imageID, Event: "failed", Message: err.Error()})
			emitSimpleError("wsl upgrade", imageID, started, err)
			return err
		}
		res := output.NewResult("wsl upgrade", imageID, started, []output.Step{{
			ID:     "upgrade",
			Name:   "candidate verify and stable swap",
			Status: "completed",
		}}, []output.Artifact{{Name: "backup", Path: backupPath}}, nil)
		emitter().EmitResult(res)
		emitter().EmitEvent(output.Event{TS: time.Now().UTC(), Level: "info", Command: "wsl upgrade", ImageID: imageID, Event: "completed"})
		return nil
	},
}

var wslRollbackCmd = &cobra.Command{
	Use:   "rollback <imageId>",
	Short: "Rollback managed WSL distro to backup",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		started := time.Now().UTC()
		imageID := args[0]
		target := wslRollbackTo
		emitter().EmitEvent(output.Event{TS: time.Now().UTC(), Level: "info", Command: "wsl rollback", ImageID: imageID, Event: "start"})
		if err := getServices().WSL.Rollback(context.Background(), wslWorkspaceFile, imageID, target, wslFallbackWindowsDir, wslNonInteractive); err != nil {
			emitter().EmitEvent(output.Event{TS: time.Now().UTC(), Level: "error", Command: "wsl rollback", ImageID: imageID, Event: "failed", Message: err.Error()})
			emitSimpleError("wsl rollback", imageID, started, err)
			return err
		}
		res := output.NewResult("wsl rollback", imageID, started, []output.Step{{
			ID:     "rollback",
			Name:   "restore stable from backup",
			Status: "completed",
		}}, nil, nil)
		emitter().EmitResult(res)
		emitter().EmitEvent(output.Event{TS: time.Now().UTC(), Level: "info", Command: "wsl rollback", ImageID: imageID, Event: "completed"})
		return nil
	},
}

var wslManagedStatusCmd = &cobra.Command{
	Use:   "status <imageId>",
	Short: "Show managed WSL status for image",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		started := time.Now().UTC()
		imageID := args[0]
		line, err := getServices().WSL.Status(context.Background(), wslWorkspaceFile, imageID)
		if err != nil {
			emitSimpleError("wsl status", imageID, started, err)
			return err
		}
		res := output.NewResult("wsl status", imageID, started, []output.Step{{
			ID:     "status",
			Name:   "query distro status",
			Status: "completed",
			Data:   map[string]interface{}{"line": line},
		}}, nil, nil)
		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(struct {
				output.Result
				Status string `json:"status"`
			}{
				Result: res,
				Status: line,
			})
		}
		emitter().EmitResult(res)
		fmt.Fprintln(cmd.OutOrStdout(), line)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(wslManagedCmd)
	wslManagedCmd.PersistentFlags().StringVar(&wslWorkspaceFile, "workspace-file", workspace.DefaultManifestPath, "Path to devcontainer superset manifest (.json)")
	wslManagedCmd.PersistentFlags().StringVar(&wslEngine, "engine", "", "Preferred build engine: docker or podman")
	wslManagedCmd.PersistentFlags().BoolVar(&wslFallbackWindowsDir, "fallback-windows-dir", false, "Fallback to windows-dir state mode on mount/elevation issues")
	wslManagedCmd.PersistentFlags().BoolVar(&wslNonInteractive, "non-interactive", false, "Disable interactive prompts and fail deterministically")

	wslManagedCmd.AddCommand(wslStateCmd)
	wslStateCmd.AddCommand(wslStateCreateCmd)

	wslManagedCmd.AddCommand(wslInstallCmd)
	wslManagedCmd.AddCommand(wslUpgradeCmd)
	wslManagedCmd.AddCommand(wslRollbackCmd)
	wslRollbackCmd.Flags().StringVar(&wslRollbackTo, "to", "", "Backup tar path or version id from metadata history")
	wslManagedCmd.AddCommand(wslManagedStatusCmd)
}
