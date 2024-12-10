package ceph

import (
	"fmt"

	microCli "github.com/canonical/microcluster/v2/client"

	"github.com/canonical/microceph/microceph/client"
)

// ExitMaintenance recover a given node from maintanence mode.
func ExitMaintenance(cli *microCli.Client, name string) error {
	// TODO: check if the node is not in maintenance mode

	// check if the node is ready to exit maintenance
	err := readyForExitMaintenance(cli, name)
	if err != nil {
		return err
	}

	// apply steps required for exiting maintenance mode
	err = applyExitMaintenanceStep(cli, name)
	if err != nil {
		return err
	}

	return nil
}

func applyExitMaintenanceStep(cli *microCli.Client, name string) error {
	// 1. start relevant snap services (mon/mds/mgr/osd)
	services, err := client.MClient.GetServices(cli)
	if err != nil {
		return nil
	}
	for _, service := range services {
		if service.Location == name {
			err = snapStart(service.Service, true)
			if err != nil {
				return err
			}
		}
	}
	// 2. bring up osds: ceph osd in
	disks, err := client.MClient.GetDisks(cli)
	for _, disk := range disks {
		if disk.Location == name {
			// bring the OSD in
			err = inOSD(disk.OSD)
			if err != nil {
				return err
			}
		}
	}
	// 3. start CRUSH to automatically rebalance: ceph osd unset noout
	// TODO: make it optional and default to true
	err = osdUnsetNoout()
	if err != nil {
		return err
	}
	// 4. switch failure domain from osd to host if needed
	needUpgrade, err := checkFailureDomainUpgrade(cli, name)
	if needUpgrade {
		err = switchFailureDomain("osd", "host")
		if err != nil {
			return err
		}
	}
	return nil
}

func readyForExitMaintenance(cli *microCli.Client, name string) error {
	// check if node exists
	err := checkNodeInCluster(cli, name)
	if err != nil {
		return err
	}
	fmt.Printf("Node '%v' is in the cluster\n", name)
	return nil
}

// checkFailureDomainDowngrade checks if the cluster need to upgrade the failure domain from
// "osd" to "host" level when the node is in maintenance mode.
func checkFailureDomainUpgrade(cli *microCli.Client, node string) (bool, error) {
	currentRule, err := getDefaultCrushRule()
	if err != nil {
		return false, err
	}
	hostRule, err := getCrushRuleID("microceph_auto_osd")
	if err != nil {
		return false, err
	}
	if currentRule != hostRule {
		// either we're at 'host' level or we're using a custom rule
		// in both cases we won't downgrade
		return false, nil
	}

	clusterMembers, err := client.MClient.GetClusterMembers(cli)
	if err != nil {
		return false, fmt.Errorf("Error getting cluster members: %v", err)
	}
	if len(clusterMembers) >= 3 {
		return true, nil
	}

	return false, nil
}
