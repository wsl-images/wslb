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
	workspaceFile   string
	workspaceAll    bool
	workspaceEngine string
)

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Manage devcontainer superset manifests",
}

var workspaceInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize .devcontainer/devcontainer.json superset manifest",
	RunE: func(cmd *cobra.Command, args []string) error {
		started := time.Now().UTC()
		path, err := workspace.InitDevcontainerFile(workspaceFile)
		if err != nil {
			emitSimpleError("workspace init", "", started, err)
			return err
		}
		emitter().EmitResult(output.NewResult("workspace init", "", started, []output.Step{{ID: "init", Name: "create manifest", Status: "completed"}}, []output.Artifact{{Name: "manifest", Path: path}}, nil))
		return nil
	},
}

var workspaceValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate workspace manifest",
	RunE: func(cmd *cobra.Command, args []string) error {
		started := time.Now().UTC()
		_, _, vr, err := getServices().Workspace.LoadAndValidate(context.Background(), workspaceFile)
		if err != nil {
			emitSimpleError("workspace validate", "", started, err)
			return err
		}
		errs := []output.Error{}
		for _, issue := range vr.Issues {
			errs = append(errs, output.Error{Code: issue.Code, Message: fmt.Sprintf("%s: %s", issue.Path, issue.Message), Remediation: issue.Remediation})
		}
		emitter().EmitResult(output.NewResult("workspace validate", "", started, []output.Step{{ID: "validate", Name: "schema and semantic validation", Status: map[bool]string{true: "completed", false: "failed"}[vr.Valid]}}, nil, errs))
		if !vr.Valid {
			return fmt.Errorf("workspace validation failed")
		}
		return nil
	},
}

var workspacePlanCmd = &cobra.Command{
	Use:   "plan [imageId]",
	Short: "Generate deterministic plan for workspace actions",
	RunE: func(cmd *cobra.Command, args []string) error {
		started := time.Now().UTC()
		imageID, err := requireArgAllOrID(workspaceAll, args)
		if err != nil {
			emitSimpleError("workspace plan", "", started, err)
			return err
		}
		plan, vr, err := getServices().Workspace.Plan(context.Background(), workspaceFile, imageID)
		if err != nil {
			errs := []output.Error{}
			for _, issue := range vr.Issues {
				errs = append(errs, output.Error{Code: issue.Code, Message: fmt.Sprintf("%s: %s", issue.Path, issue.Message), Remediation: issue.Remediation})
			}
			if len(errs) == 0 {
				errs = append(errs, output.Error{Code: "WSLB_PLAN_FAILED", Message: err.Error()})
			}
			emitter().EmitResult(output.NewResult("workspace plan", imageID, started, nil, nil, errs))
			return err
		}

		res := output.NewResult("workspace plan", imageID, started, []output.Step{{ID: "plan", Name: "build workspace plan", Status: "completed", Data: map[string]interface{}{"imageCount": len(plan.Images)}}}, nil, nil)
		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(struct {
				output.Result
				Plan workspace.Plan `json:"plan"`
			}{Result: res, Plan: plan})
		}
		emitter().EmitResult(res)
		for _, ip := range plan.Images {
			fmt.Printf("%s (%s):\n", ip.ImageID, ip.Target)
			for _, st := range ip.Steps {
				fmt.Printf("  - %s\n", st)
			}
		}
		return nil
	},
}

var workspaceBuildCmd = &cobra.Command{
	Use:   "build [imageId]",
	Short: "Build workspace images",
	RunE: func(cmd *cobra.Command, args []string) error {
		started := time.Now().UTC()
		imageID, err := requireArgAllOrID(workspaceAll, args)
		if err != nil {
			emitSimpleError("workspace build", "", started, err)
			return err
		}
		emitter().EmitEvent(output.Event{TS: time.Now().UTC(), Level: "info", Command: "workspace build", ImageID: imageID, Event: "start"})
		if err := getServices().Workspace.Build(context.Background(), workspaceFile, imageID, workspaceEngine); err != nil {
			emitter().EmitEvent(output.Event{TS: time.Now().UTC(), Level: "error", Command: "workspace build", ImageID: imageID, Event: "failed", Message: err.Error()})
			emitSimpleError("workspace build", imageID, started, err)
			return err
		}
		emitter().EmitResult(output.NewResult("workspace build", imageID, started, []output.Step{{ID: "build", Name: "build images", Status: "completed"}}, nil, nil))
		emitter().EmitEvent(output.Event{TS: time.Now().UTC(), Level: "info", Command: "workspace build", ImageID: imageID, Event: "completed"})
		return nil
	},
}

var workspacePublishCmd = &cobra.Command{
	Use:   "publish [imageId]",
	Short: "Publish workspace images",
	RunE: func(cmd *cobra.Command, args []string) error {
		started := time.Now().UTC()
		imageID, err := requireArgAllOrID(workspaceAll, args)
		if err != nil {
			emitSimpleError("workspace publish", "", started, err)
			return err
		}
		if err := getServices().Workspace.Publish(context.Background(), workspaceFile, imageID); err != nil {
			emitSimpleError("workspace publish", imageID, started, err)
			return err
		}
		emitter().EmitResult(output.NewResult("workspace publish", imageID, started, []output.Step{{ID: "publish", Name: "publish images", Status: "completed"}}, nil, nil))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(workspaceCmd)
	workspaceCmd.PersistentFlags().StringVar(&workspaceFile, "workspace-file", workspace.DefaultManifestPath, "Path to devcontainer superset manifest (.json)")

	workspaceCmd.AddCommand(workspaceInitCmd)
	workspaceCmd.AddCommand(workspaceValidateCmd)
	workspaceCmd.AddCommand(workspacePlanCmd)
	workspaceCmd.AddCommand(workspaceBuildCmd)
	workspaceCmd.AddCommand(workspacePublishCmd)

	workspacePlanCmd.Flags().BoolVar(&workspaceAll, "all", false, "Operate on all images")
	workspaceBuildCmd.Flags().BoolVar(&workspaceAll, "all", false, "Operate on all images")
	workspacePublishCmd.Flags().BoolVar(&workspaceAll, "all", false, "Operate on all images")
	workspaceBuildCmd.Flags().StringVar(&workspaceEngine, "engine", "", "Preferred build engine: docker or podman")
}
