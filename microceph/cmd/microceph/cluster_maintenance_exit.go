package main

import (
	// "context"
	"fmt"

	// "github.com/canonical/microcluster/v2/microcluster"
	"github.com/spf13/cobra"

	// "github.com/canonical/microceph/microceph/api/types"
	// "github.com/canonical/microceph/microceph/ceph"
	// "github.com/canonical/microceph/microceph/client"
)

type cmdClusterMaintenanceExit struct {
	common       *CmdControl

	flagForce bool
}

func (c *cmdClusterMaintenanceExit) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exit <NAME>",
		Short: "Recover a given server from maintenance mode.",
		RunE:  c.Run,
	}

	cmd.Flags().BoolVarP(&c.flagForce, "force", "f", true, "Forcibly recover the server from maintenance mode.")
	return cmd
}

func (c *cmdClusterMaintenanceExit) Run(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return cmd.Help()
	}

	// m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir})
	// if err != nil {
	// 	return fmt.Errorf("unable to configure MicroCeph: %w", err)
	// }

	// cli, err := m.LocalClient()
	// if err != nil {
	// 	return err
	// }

	fmt.Println("Called `microceph ceph cluster maintenance exit`.")

	// TODO
	return nil
}
