package main

import (
	"github.com/canonical/microcluster/v2/microcluster"
	"github.com/spf13/cobra"

	"github.com/canonical/microceph/microceph/ceph"
	"github.com/canonical/microceph/microceph/client"
)

type cmdClusterMaintenanceEnter struct {
	common *CmdControl

	flagConfirm                       bool
	flagStopOsds                      bool
	flagSetNoout                      bool
	flagBypassSafetyChecks            bool
	flagConfirmFailureDomainDowngrade bool
}

func (c *cmdClusterMaintenanceEnter) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enter <NAME>",
		Short: "Enter maintenance mode.",
		RunE:  c.Run,
	}

	cmd.Flags().BoolVar(&c.flagConfirm, "yes-i-really-mean-it", false, "Confirm entering maintenance mode")
	cmd.Flags().BoolVar(&c.flagStopOsds, "stop-osds", false, "Optionally stop the OSDs on this node")
	cmd.Flags().BoolVar(&c.flagSetNoout, "set-noout", true, "Optionally set noout during maintenance mode")
	cmd.Flags().BoolVar(&c.flagBypassSafetyChecks, "bypass-safety-checks", false, "Bypass safety checks")
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

	clusterClient, err := m.LocalClient()
	if err != nil {
		return err
	}

	config := ceph.EnterMaintenanceConfig{
		c.flagConfirm,
		c.flagStopOsds,
		c.flagSetNoout,
		c.flagBypassSafetyChecks,
		c.flagConfirmFailureDomainDowngrade,
	}

	err = ceph.EnterMaintenance(clusterClient, client.MClient, args[0], config)
	if err != nil {
		return err
	}

	return nil
}
