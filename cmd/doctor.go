package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/wsl-images/wslb/internal/output"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check local prerequisites and environment health",
	RunE: func(cmd *cobra.Command, args []string) error {
		started := time.Now().UTC()
		report := getServices().Doctor.Run(context.Background())

		failures := 0
		for _, c := range report.Checks {
			if c.Status == "fail" {
				failures++
			}
		}

		steps := []output.Step{{
			ID:     "doctor",
			Name:   "run prerequisite checks",
			Status: map[bool]string{true: "completed", false: "failed"}[report.OK],
			Data: map[string]interface{}{
				"checkCount": len(report.Checks),
				"failCount":  failures,
			},
		}}
		errs := []output.Error{}
		if !report.OK {
			errs = append(errs, output.Error{
				Code:    "WSLB_DOCTOR_FAILED",
				Message: "one or more required checks failed",
			})
		}
		res := output.NewResult("doctor", "", started, steps, nil, errs)

		if jsonOut {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(struct {
				output.Result
				Checks interface{} `json:"checks"`
			}{
				Result: res,
				Checks: report.Checks,
			})
		}

		emitter().EmitResult(res)
		for _, c := range report.Checks {
			line := fmt.Sprintf("- [%s] %s: %s", c.Status, c.ID, c.Message)
			if c.Remediation != "" {
				line += "\n    remediation: " + c.Remediation
			}
			fmt.Fprintln(cmd.OutOrStdout(), line)
		}
		if !report.OK {
			return fmt.Errorf("doctor checks failed")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
