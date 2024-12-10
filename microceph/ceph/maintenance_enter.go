package ceph

import (
	"fmt"

	microCli "github.com/canonical/microcluster/v2/client"

	"github.com/canonical/microceph/microceph/client"
)

// EnterMaintenance put a given node into maintanence mode.
func EnterMaintenance(cli *microCli.Client, name string, bypassSafety, confirmFailureDomainDowngrade bool) error {
	// TODO: check if the node is already in maintenance mode

	// check if the node is ready for maintenance
	err := readyForEnterMaintenance(cli, name, bypassSafety, confirmFailureDomainDowngrade)
	if err != nil {
		return err
	}

	// apply steps required for entering maintenance mode
	err = applyEnterMaintenanceStep(cli, name)
	if err != nil {
		return err
	}

	return nil
}

func applyEnterMaintenanceStep(cli *microCli.Client, name string) error {
	// 1. switch failure domain from host to osd if needed
	needDowngrade, err := checkFailureDomainDowngrade(cli, name)
	if needDowngrade {
		err = switchFailureDomain("host", "osd")
		if err != nil {
			return err
		}
	}
	// 2. stop CRUSH to automatically rebalance: ceph osd set noout
	// TODO: make it optional and default to true
	err = osdSetNoout()
	if err != nil {
		return err
	}
	// 3. check again if osd is okay-to-stop, and stop and kill the OSDs if it's okay
	disks, err := client.MClient.GetDisks(cli)
	for _, disk := range disks {
		if disk.Location == name && safetyCheckStop(disk.OSD) != nil {
			// take the OSD out and down
			err = outDownOSD(disk.OSD)
			if err != nil {
				return err
			}
			// // kill the OSD service idempotently
			// _ = killOSD(disk.OSD)
		}
	}
	// 4. stop the relevant snap services (mon/mds/mgr/osd)
	services, err := client.MClient.GetServices(cli)
	if err != nil {
		return nil
	}
	for _, service := range services {
		if service.Location == name {
			err = snapStop(service.Service, true)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func readyForEnterMaintenance(cli *microCli.Client, name string, bypassSafety, confirmFailureDomainDowngrade bool) error {
	// check if node exists
	err := checkNodeInCluster(cli, name)
	if err != nil {
		return err
	}
	fmt.Printf("Node '%v' is in the cluster\n", name)

	if !bypassSafety {
		// check if OSDs are enough for maintenance mode
		// need at least 3 OSDs
		err = checkMinimumOSDs(cli, name, 3)
		if err != nil {
			return err
		}
		fmt.Println("Remaining OSDs (>=3) is enough for the cluster")

		// check if the OSDs are okay to stop
		err = checkOkayToStopOSDs(cli, name)
		if err != nil {
			return err
		}
		fmt.Printf("All OSDs in node '%v' is okay to stop\n", name)

		// check if non-OSDs services (mon/mds/mgr) are enough for maintenance mode
		// need at least (3,1,1) for (mon/mds/mgr)
		err = checkMinimumNonOSDServices(cli, name, 3, 1, 1)
		if err != nil {
			return err
		}
		fmt.Println("Remaining mon/mds/mgr (>= 3/1/1) is enough for the cluster")
	} else {
		fmt.Println("Bypassed safety checks before entering maintenance mode")
	}

	// check if downgrade of failure domain is needed or not for maintenance mode
	needDowngrade, err := checkFailureDomainDowngrade(cli, name)
	if err != nil {
		return err
	}
	fmt.Printf("Required downgrading failure domain from 'host' to 'osd': %v\n", needDowngrade)
	if needDowngrade && !confirmFailureDomainDowngrade {
		return fmt.Errorf("Failure domain downgrade is required, use --confirm-failure-domain-downgrade to confirm")
	}

	return nil
}

// checkNodeInCluster checks if the given node is in the cluster.
func checkNodeInCluster(cli *microCli.Client, name string) error {
	clusterMembers, err := client.MClient.GetClusterMembers(cli)
	if err != nil {
		return fmt.Errorf("Error getting cluster members: %v", err)
	}

	for _, member := range clusterMembers {
		if member == name {
			// found the node, exit here
			return nil
		}
	}

	return fmt.Errorf("Node %v not found", name)
}

// checkMinimumOSDs checks if the cluster has at least x number of OSDs after the node enters
// maintanence mode.
func checkMinimumOSDs(cli *microCli.Client, node string, x int) error {
	// TODO: count only in and up OSDs?

	disks, err := client.MClient.GetDisks(cli)
	if err != nil {
		return fmt.Errorf("Error getting disks: %v", err)
	}

	for _, disk := range disks {
		if disk.Location == node && len(disks) <= x {
			// osd is found but the total number of osds is not enough to make the cluster remain
			// healthy if the node enters maintenence mode.
			return fmt.Errorf("There is not enough remaining OSDs after node %v enter maintenance mode.", node)
		}
	}

	return nil
}

// checkMinimumNonOSDServices checks if the cluster has at least x/y/z number of mon/mds/mgr after
// the node enters maintanence mode.
func checkMinimumNonOSDServices(cli *microCli.Client, node string, x, y, z int) error {
	// TODO: count only active service ?

	services, err := client.MClient.GetServices(cli)
	if err != nil {
		return fmt.Errorf("Error getting services: %v", err)
	}

	// this is the service map **excluding** the service on the node
	serviceMap := map[string]int{
		"mon": 0,
		"mgr": 0,
		"mds": 0,
	}
	for _, service := range services {
		if service.Location != node {
			serviceMap[service.Service]++
		}
	}

	// the remaining services must be sufficient to make the cluster healthy after the node enters
	// maintanence mode.
	if serviceMap["mon"] < x || serviceMap["mgr"] < y || serviceMap["mds"] < z {
		return fmt.Errorf("Need at least %v mon, %v mds, and %v mgr besides %v", x, y, z, node)
	}

	return nil
}

// checkFailureDomainDowngrade checks if the cluster need to downgrade the failure domain from
// "host" to "osd" level when the node is in maintenance mode.
func checkFailureDomainDowngrade(cli *microCli.Client, node string) (bool, error) {
	currentRule, err := getDefaultCrushRule()
	if err != nil {
		return false, err
	}
	hostRule, err := getCrushRuleID("microceph_auto_host")
	if err != nil {
		return false, err
	}
	if currentRule != hostRule {
		// either we're at 'osd' level or we're using a custom rule
		// in both cases we won't downgrade
		return false, nil
	}

	clusterMembers, err := client.MClient.GetClusterMembers(cli)
	if err != nil {
		return false, fmt.Errorf("Error getting cluster members: %v", err)
	}
	// putting the node in maintenance mode will stop the osds and other services, it's like
	// temporarily "removing" a node
	if len(clusterMembers) <= 3 {
		return true, nil
	}

	return false, nil
}

// checkOkayToStopOSDs checks if the OSDs in the node can be safely stopped when the node enters
// maintanence mode.
func checkOkayToStopOSDs(cli *microCli.Client, node string) error {
	disks, err := client.MClient.GetDisks(cli)
	if err != nil {
		return fmt.Errorf("Error getting disks: %v", err)
	}

	failedToStopOSDs := []int64{}
	for _, disk := range disks {
		if disk.Location == node && safetyCheckStop(disk.OSD) != nil {
			failedToStopOSDs = append(failedToStopOSDs, disk.OSD)
		}
	}

	if len(failedToStopOSDs) > 0 {
		return fmt.Errorf("OSDs %v cannot be safely stopped", failedToStopOSDs)
	}

	return nil
}
