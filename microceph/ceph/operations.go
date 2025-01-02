package ceph

import (
	"fmt"

	"github.com/canonical/lxd/shared/logger"

	microCli "github.com/canonical/microcluster/v2/client"

	"github.com/canonical/microceph/microceph/client"
)

// RunOperations runs the provided operations or prints out the action plan.
func RunOperations(name string, operations []Operation, dryRun bool) error {
	for _, ops := range operations {
		if dryRun {
			fmt.Println(ops.DryRun(name))
		} else {
			err := ops.Run(name)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Operation is a interface for ceph and microceph operations.
type Operation interface {
	// Run executes the operation and return the error if any.
	Run(string) error

	// DryRun returns the string representation of the operation.
	DryRun(string) string
}

// ClusterOps is the base struct for all operations.
type ClusterOps struct {
	CephClient    client.ClientInterface
	ClusterClient *microCli.Client

}

// CheckNodeInClusterOps is an operation to check if a node is in the microceph cluster.
type CheckNodeInClusterOps struct {
	ClusterOps
}

// Run checks if a node is in the microceph cluster.
func (o *CheckNodeInClusterOps) Run(name string) error {
	clusterMembers, err := o.CephClient.GetClusterMembers(o.ClusterClient)
	if err != nil {
		return fmt.Errorf("error getting cluster members: %v", err)
	}

	for _, member := range clusterMembers {
		if member == name {
			logger.Infof("node '%s' is in the cluster.", name)
			return nil
		}
	}

	return fmt.Errorf("node '%s' not found", name)
}

// DryRun prints out the action plan.
func (o *CheckNodeInClusterOps) DryRun(name string) string {
	return fmt.Sprintf("Check if node '%s' is in the cluster.", name)
}

// CheckOsdOkToStopOps is an operation to check if osds in a node are ok-to-stop.
type CheckOsdOkToStopOps struct {
	ClusterOps
}

// Run checks osds in a node are ok-to-stop.
func (o *CheckOsdOkToStopOps) Run(name string) error {
	disks, err := o.CephClient.GetDisks(o.ClusterClient)
	if err != nil {
		return fmt.Errorf("error getting disks: %v", err)
	}

	OsdsToCheck := []int64{}
	for _, disk := range disks {
		if disk.Location == name {
			OsdsToCheck = append(OsdsToCheck, disk.OSD)
		}
	}

	if !testSafeStop(OsdsToCheck) {
		return fmt.Errorf("osd.%v cannot be safely stopped", OsdsToCheck)
	}

	logger.Infof("osd.%v can be safely stopped.", OsdsToCheck)
	return nil
}

// DryRun prints out the action plan.
func (o *CheckOsdOkToStopOps) DryRun(name string) string {
	return fmt.Sprintf("Check if osds in node '%s' are ok-to-stop.", name)
}

// CheckNonOsdSvcEnoughOps is an operation to check if non-osd service in a node are enough.
type CheckNonOsdSvcEnoughOps struct {
	ClusterOps

	MinMon int
	MinMds int
	MinMgr int
}

// Run checks if non-osds service in a node are enough.
func (o *CheckNonOsdSvcEnoughOps) Run(name string) error {
	services, err := o.CephClient.GetServices(o.ClusterClient)
	if err != nil {
		return fmt.Errorf("error getting services: %v", err)
	}

	remains := map[string]int{
		"mon": 0,
		"mgr": 0,
		"mds": 0,
	}
	for _, service := range services {
		// do not count the service on this node
		if service.Location != name {
			remains[service.Service]++
		}
	}

	// the remaining services must be sufficient to make the cluster healthy after the node enters
	// maintanence mode.
	if remains["mon"] < o.MinMon || remains["mds"] < o.MinMds || remains["mgr"] < o.MinMgr {
		return fmt.Errorf("need at least %d mon, %d mds, and %d mgr services in the cluster besides those in node '%s'", o.MinMon, o.MinMds, o.MinMgr, name)
	}
	logger.Infof("remaining mon (%d), mds (%d), and mgr (%d) services in the cluster are enough after '%s' enters maintenance mode", remains["mon"], remains["mds"], remains["mgr"], name)

	return nil
}

// DryRun prints out the action plan.
func (o *CheckNonOsdSvcEnoughOps) DryRun(name string) string {
	return fmt.Sprintf("Check if there are at least %d mon, %d mds, and %d mgr services in the cluster besides those in node '%s'", o.MinMon, o.MinMds, o.MinMgr, name)
}

// SetNooutOps is an operation to set noout for the ceph cluster.
type SetNooutOps struct{
	ClusterOps
}

// Run `ceph osd set noout` for the ceph cluster.
func (o *SetNooutOps) Run(name string) error {
	err := setOsdNooutFlag(true)
	if err != nil {
		return err
	}
	return nil
}

// DryRun prints out the action plan.
func (o *SetNooutOps) DryRun(name string) string {
	return "Run `ceph osd set noout`."
}

// AssertNooutFlagSetOps is an operation to assert noout has been set for the ceph cluster.
type AssertNooutFlagSetOps struct{
	ClusterOps
}

// Run asserts noout has been set for the ceph cluster.
func (o *AssertNooutFlagSetOps) Run(name string) error {
	set, err := isOsdNooutSet()
	if err != nil {
		return err
	}
	if !set {
		return fmt.Errorf("osd has 'noout' flag unset.")
	}
	logger.Info("osd has 'noout' flag set.")
	return nil
}

// DryRun prints out the action plan.
func (o *AssertNooutFlagSetOps) DryRun(name string) string {
	return "Assert osd has 'noout' flag set."
}

// AssertNooutFlagUnsetOps is an operation to assert noout has been unset for the ceph cluster.
type AssertNooutFlagUnsetOps struct{
	ClusterOps
}

// Run asserts noout has been unset for the ceph cluster.
func (o *AssertNooutFlagUnsetOps) Run(name string) error {
	set, err := isOsdNooutSet()
	if err != nil {
		return err
	}
	if set {
		return fmt.Errorf("osd has 'noout' flag set.")
	}
	logger.Info("osd has 'noout' flag unset.")
	return nil
}

// DryRun prints out the action plan.
func (o *AssertNooutFlagUnsetOps) DryRun(name string) string {
	return "Assert osd has 'noout' flag unset."
}

// StopOsdOps is an operation to stop osd service for a node.
type StopOsdOps struct {
	ClusterOps
}

// Run stops the osd service for a node.
func (o *StopOsdOps) Run(name string) error {
	err := o.CephClient.PutOsds(o.ClusterClient, false, name)
	if err != nil {
		logger.Errorf("unable to stop osd service in node '%s': %v", name, err)
		return err
	}
	logger.Infof("stopped osd service in node '%s'.", name)
	return nil
}

// DryRun prints out the action plan.
func (o *StopOsdOps) DryRun(name string) string {
	return fmt.Sprintf("Stop osd service in node '%s'.", name)
}

// StartOsdOps is an operation to start osd service for a node.
type StartOsdOps struct {
	ClusterOps
}

// Run starts the osd service for a node.
func (o *StartOsdOps) Run(name string) error {
	err := o.CephClient.PutOsds(o.ClusterClient, true, name)
	if err != nil {
		logger.Errorf("unable to start osd service in node '%s': %v", name, err)
		return err
	}
	logger.Infof("started osd service in node '%s'", name)
	return nil
}

// DryRun prints out the action plan.
func (o *StartOsdOps) DryRun(name string) string {
	return fmt.Sprintf("Start osd service in node '%s'.", name)
}

// UnsetNooutOps is an operation to unset noout for the ceph cluster.
type UnsetNooutOps struct{
	ClusterOps
}

// Run `ceph osd unset noout` for the ceph cluster.
func (o *UnsetNooutOps) Run(name string) error {
	err := setOsdNooutFlag(false)
	if err != nil {
		return err
	}
	logger.Info("unset osd noout.")
	return nil
}

// DryRun prints out the action plan.
func (o *UnsetNooutOps) DryRun(name string) string {
	return "Run `ceph osd unset noout`."
}
