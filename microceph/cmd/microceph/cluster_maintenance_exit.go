package main

import (
	"fmt"
	"github.com/canonical/microcluster/v2/microcluster"
	"github.com/spf13/cobra"

	"github.com/canonical/microceph/microceph/ceph"
	"github.com/canonical/microceph/microceph/client"
)

type cmdClusterMaintenanceExit struct {
	common *CmdControl

	flagDryRun bool
}

func (c *cmdClusterMaintenanceExit) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exit <NAME>",
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

	name := args[0]
	clusterOps := ceph.ClusterOps{CephClient: client.MClient, ClusterClient: cli}
	operations := []ceph.Operation{
		&ceph.CheckNodeInClusterOps{ClusterOps: clusterOps},
	}

	// idempotently unset noout and start osd service
	operations = append(operations, []ceph.Operation{
		&ceph.UnsetNooutOps{ClusterOps: clusterOps},
		&ceph.AssertNooutFlagUnsetOps{ClusterOps: clusterOps},
		&ceph.StartOsdOps{ClusterOps: clusterOps},
	}...)

	err = ceph.RunOperations(name, operations, c.flagDryRun, false)
	if err != nil {
		return fmt.Errorf("failed to exit maintenance mode: %v", err)
	}

	return nil
}
