package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/wsl-images/wslb/internal/output"
)

var catalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Feature catalog operations",
}

var catalogFeatureCmd = &cobra.Command{
	Use:   "feature",
	Short: "Feature metadata operations",
}

var catalogFeatureDescribeCmd = &cobra.Command{
	Use:   "describe <ref>",
	Short: "Describe a feature reference",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		started := time.Now().UTC()
		ref := args[0]
		meta, err := getServices().Catalog.DescribeFeature(context.Background(), ref)
		if err != nil {
			emitSimpleError("catalog feature describe", "", started, err)
			return err
		}

		res := output.NewResult("catalog feature describe", "", started, []output.Step{{
			ID:     "describe",
			Name:   "resolve feature metadata",
			Status: "completed",
			Data: map[string]interface{}{
				"ref": ref,
			},
		}}, nil, nil)

		if jsonOut {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(struct {
				output.Result
				Feature interface{} `json:"feature"`
			}{
				Result:  res,
				Feature: meta,
			})
		}

		emitter().EmitResult(res)
		fmt.Fprintf(cmd.OutOrStdout(), "Feature: %s\nName: %s\nVersion: %s\n", meta.ID, meta.Name, meta.Version)
		if meta.Description != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Description: %s\n", meta.Description)
		}
		if len(meta.Options) > 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "Options:")
			for k, v := range meta.Options {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %s (%s)\n", k, v.Type)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(catalogCmd)
	catalogCmd.AddCommand(catalogFeatureCmd)
	catalogFeatureCmd.AddCommand(catalogFeatureDescribeCmd)
}
