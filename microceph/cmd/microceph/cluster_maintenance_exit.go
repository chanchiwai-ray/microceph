package main

import (
	"github.com/canonical/microcluster/v2/microcluster"
	"github.com/spf13/cobra"

	"github.com/canonical/microceph/microceph/ceph"
	"github.com/canonical/microceph/microceph/client"
)

type cmdClusterMaintenanceExit struct {
	common *CmdControl

	flagConfirm                     bool
	flagUnsetNoout                  bool
	flagBypassSafetyChecks          bool
	flagConfirmFailureDomainUpgrade bool
}

func (c *cmdClusterMaintenanceExit) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exit <NAME>",
		Short: "Exit maintenance mode.",
		RunE:  c.Run,
	}

	cmd.Flags().BoolVar(&c.flagConfirm, "yes-i-really-mean-it", false, "Confirm exiting maintenance mode")
	cmd.Flags().BoolVar(&c.flagUnsetNoout, "unset-noout", true, "Optionally unset noout during maintenance mode")
	cmd.Flags().BoolVar(&c.flagBypassSafetyChecks, "bypass-safety-checks", false, "Bypass safety checks")
	cmd.Flags().BoolVar(&c.flagConfirmFailureDomainUpgrade, "confirm-failure-domain-upgrade", false, "Confirm failure domain upgrade if required")

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

	clusterClient, err := m.LocalClient()
	if err != nil {
		return err
	}

	config := ceph.ExitMaintenanceConfig{
		c.flagConfirm,
		c.flagUnsetNoout,
		c.flagBypassSafetyChecks,
		c.flagConfirmFailureDomainUpgrade,
	}

	err = ceph.ExitMaintenance(clusterClient, client.MClient, args[0], config)
	if err != nil {
		return err
	}

	return nil
}
