package main

import (
	"github.com/canonical/microcluster/v2/microcluster"
	"github.com/spf13/cobra"

	"github.com/canonical/microceph/microceph/ceph"
)

type cmdClusterMaintenanceExit struct {
	common       *CmdControl

	flagForce bool
	flagConfirmFailureDomainUpgrade bool
}

func (c *cmdClusterMaintenanceExit) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exit <NAME>",
		Short: "Recover a given server from maintenance mode.",
		RunE:  c.Run,
	}

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

	err = ceph.ExitMaintenance(cli, args[0])
	if err != nil {
		return err
	}

	return nil
}
