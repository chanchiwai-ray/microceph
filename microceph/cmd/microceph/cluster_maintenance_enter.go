package main

import (
	"context"
	"fmt"
	"github.com/canonical/microcluster/v2/microcluster"
	"github.com/spf13/cobra"
	"strings"

	"github.com/canonical/microceph/microceph/client"
)

type cmdClusterMaintenanceEnter struct {
	common *CmdControl

	flagForce     bool
	flagDryRun    bool
	flagSetNoout  bool
	flagStopOsds  bool
	flagCheckOnly bool
}

func (c *cmdClusterMaintenanceEnter) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enter <NODE_NAME>",
		Short: "Enter maintenance mode.",
		RunE:  c.Run,
	}

	cmd.Flags().BoolVar(&c.flagForce, "force", false, "Force to enter maintenance mode.")
	cmd.Flags().BoolVar(&c.flagDryRun, "dry-run", false, "Dry run the command.")
	cmd.Flags().BoolVar(&c.flagSetNoout, "set-noout", true, "Stop CRUSH from rebalancing the cluster.")
	cmd.Flags().BoolVar(&c.flagStopOsds, "stop-osds", false, "Stop the OSDS when entering maintenance mode.")
	cmd.Flags().BoolVar(&c.flagCheckOnly, "check-only", false, "Only run the preflight checks.")
	return cmd
}

func (c *cmdClusterMaintenanceEnter) Run(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return cmd.Help()
	}

	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir})
	if err != nil {
		return err
	}

	cli, err := m.LocalClient()
	if err != nil {
		return err
	}

	results, err := client.EnterMaintenance(context.Background(), cli, args[0], c.flagForce, c.flagDryRun, c.flagSetNoout, c.flagStopOsds, c.flagCheckOnly)
	if err != nil {
		return fmt.Errorf("failed to enter maintenance mode: %v", err)
	}

	errMessages := []string{}
	for _, result := range results {
		if c.flagDryRun {
			fmt.Println(result.Action)
		} else {
			errMessage := result.Error
			if errMessage == "" {
				fmt.Printf("%s (passed)\n", result.Action)
			} else {
				errMessages = append(errMessages, fmt.Sprintf("(%s)", errMessage))
				fmt.Printf("%s (failed: %s)\n", result.Action, errMessage)
			}
		}
	}

	// Return the error messages if there are errors and is not forced
	if len(errMessages) != 0 && !c.flagForce {
		return fmt.Errorf("[%s]", strings.Join(errMessages, " "))
	}

	return nil
}
