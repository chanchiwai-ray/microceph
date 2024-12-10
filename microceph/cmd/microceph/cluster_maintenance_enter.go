package main

import (
	"github.com/canonical/microcluster/v2/microcluster"
	"github.com/spf13/cobra"

	"github.com/canonical/microceph/microceph/ceph"
)

type cmdClusterMaintenanceEnter struct {
	common       *CmdControl

	flagBypassSafety bool
	flagConfirmFailureDomainDowngrade bool
}

func (c *cmdClusterMaintenanceEnter) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enter <NAME> [--bypass-safety-checks=false] [--confirm-failure-domain-downgrade=false]",
		Short: "Put a given server into maintenance mode.",
		RunE:  c.Run,
	}

	cmd.Flags().BoolVar(&c.flagBypassSafety, "bypass-safety-checks", false, "Bypass safety checks")
	cmd.Flags().BoolVar(&c.flagConfirmFailureDomainDowngrade, "confirm-failure-domain-downgrade", false, "Confirm failure domain downgrade if required")
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

	err = ceph.EnterMaintenance(cli, args[0], c.flagBypassSafety, c.flagConfirmFailureDomainDowngrade)
	if err != nil {
		return err
	}

	return nil
}
