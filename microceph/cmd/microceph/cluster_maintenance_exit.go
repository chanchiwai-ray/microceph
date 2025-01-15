package main

import (
	"context"
	"fmt"
	"github.com/canonical/microcluster/v2/microcluster"
	"github.com/spf13/cobra"
	"strings"

	"github.com/canonical/microceph/microceph/client"
)

type cmdClusterMaintenanceExit struct {
	common *CmdControl

	flagDryRun bool
}

func (c *cmdClusterMaintenanceExit) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exit <NODE_NAME>",
		Short: "Exit maintenance mode.",
		RunE:  c.Run,
	}

	cmd.Flags().BoolVar(&c.flagDryRun, "dry-run", false, "Dry run the command.")

	return cmd
}

func (c *cmdClusterMaintenanceExit) Run(cmd *cobra.Command, args []string) error {
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

	plan, err := client.ExitMaintenance(context.Background(), cli, args[0], c.flagDryRun)
	if err != nil {
		return fmt.Errorf("failed to enter maintenance mode: %v", err)
	}
	if len(plan) != 0 {
		fmt.Println(strings.Join(plan, "\n"))
	}

	return nil
}
