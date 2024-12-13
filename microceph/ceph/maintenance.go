package ceph

import (
	"context"
	"fmt"

	microCli "github.com/canonical/microcluster/v2/client"

	"github.com/canonical/microceph/microceph/client"
)

type EnterMaintenanceConfig struct {
	Confirm                       bool
	StopOsds                      bool
	SetNoout                      bool
	BypassSafetyCheck             bool
	ConfirmFailureDomainDowngrade bool
}

// EnterMaintenance put a given node into maintanence mode.
func EnterMaintenance(clusterClient *microCli.Client, cephClient client.ClientInterface, n string, c EnterMaintenanceConfig) error {
	// TODO: check if the node is in maintenance mode

	ops := []Operation{}

	// safety checks
	if !c.BypassSafetyCheck {
		ops = append(ops, []Operation{
			&isNodeInClusterOps{cephClient, clusterClient},
			&isOsdEnoughOps{cephClient, clusterClient, 3},
			&isOkayToStopOsdOps{cephClient, clusterClient},
			&isNonOsdServiceEnoughOps{cephClient, clusterClient, 3, 1, 1},
		}...)
	}

	// ops to enter maintenance mode
	ops = append(ops, []Operation{
		&downgradeFailureDomainOps{cephClient, clusterClient, c.ConfirmFailureDomainDowngrade},
		&osdSetNooutOps{c.SetNoout},
		&stopOsdOps{cephClient, clusterClient, c.StopOsds},
		&stopNonOsdOps{cephClient, clusterClient},
	}...)

	// execute plan
	m := maintenance{n}
	err := m.Run(ops, !c.Confirm)
	if err != nil {
		return fmt.Errorf("failed to enter maintenance mode: %v", err)
	}
	return nil
}

type ExitMaintenanceConfig struct {
	Confirm                     bool
	UnsetNoout                  bool
	BypassSafetyCheck           bool
	ConfirmFailureDomainUpgrade bool
}

// ExitMaintenance recover a given node from maintanence mode.
func ExitMaintenance(clusterClient *microCli.Client, cephClient client.ClientInterface, n string, c ExitMaintenanceConfig) error {
	// TODO: check if the node is not in maintenance mode

	ops := []Operation{}
	// safety checks
	if !c.BypassSafetyCheck {
		ops = append(ops, []Operation{
			&isNodeInClusterOps{cephClient, clusterClient},
		}...)
	}
	// ops to exit maintenance mode
	ops = append(ops, []Operation{
		&startNonOsdOps{cephClient, clusterClient},
		&startOsdOps{cephClient, clusterClient},
		&osdUnsetNooutOps{c.UnsetNoout},
		&upgradeFailureDomainOps{cephClient, clusterClient, c.ConfirmFailureDomainUpgrade},
	}...)
	// execute plan
	m := maintenance{n}
	err := m.Run(ops, !c.Confirm)
	if err != nil {
		return fmt.Errorf("failed to exit maintenance mode: %v", err)
	}
	return nil
}

type opsRunner interface {
	Run(operations []Operation, dryRun bool) error
}

type maintenance struct {
	nodeName string
}

func (m *maintenance) Run(operations []Operation, dryRun bool) error {
	for _, ops := range operations {
		if dryRun {
			fmt.Println(ops.DryRun(m.nodeName))
		} else {
			err := ops.Run(m.nodeName)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

//
// Operations
//

type Operation interface {
	Run(string) error
	DryRun(string) string
}

type isNodeInClusterOps struct {
	cephClient    client.ClientInterface
	clusterClient *microCli.Client
}

func (o *isNodeInClusterOps) Run(name string) error {
	clusterMembers, err := o.cephClient.GetClusterMembers(o.clusterClient)
	if err != nil {
		return fmt.Errorf("Error getting cluster members: %v", err)
	}

	for _, member := range clusterMembers {
		if member == name {
			fmt.Printf("Node '%s' is in the cluster.\n", name)
			return nil
		}
	}

	return fmt.Errorf("Node %s not found", name)
}

func (o *isNodeInClusterOps) DryRun(name string) string {
	return fmt.Sprintf("Will check if node '%s' is in the cluster.", name)
}

type isOsdEnoughOps struct {
	cephClient    client.ClientInterface
	clusterClient *microCli.Client

	minOsd int
}

func (o *isOsdEnoughOps) Run(name string) error {
	// TODO: count only in and up OSDs?
	disks, err := o.cephClient.GetDisks(o.clusterClient)
	if err != nil {
		return fmt.Errorf("Error getting disks: %v", err)
	}

	remains := 0
	for _, disk := range disks {
		if disk.Location != name {
			remains++
		}
	}

	if remains < o.minOsd {
		return fmt.Errorf("Need at least %d osds in the cluster besides those in the node '%s'.", o.minOsd, name)
	}
	fmt.Printf("Remaining osds (%d) in the cluster is enough after '%s' enters maintenance mode\n", remains, name)

	return nil
}

func (o *isOsdEnoughOps) DryRun(name string) string {
	return fmt.Sprintf("Will check if node '%s' has at least %d osds in the cluster.", name, o.minOsd)
}

type isNonOsdServiceEnoughOps struct {
	cephClient    client.ClientInterface
	clusterClient *microCli.Client

	minMon int
	minMds int
	minMgr int
}

func (o *isNonOsdServiceEnoughOps) Run(name string) error {
	// TODO: count only active service ?

	services, err := o.cephClient.GetServices(o.clusterClient)
	if err != nil {
		return fmt.Errorf("Error getting services: %v", err)
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
	if remains["mon"] < o.minMon || remains["mds"] < o.minMds || remains["mgr"] < o.minMgr {
		return fmt.Errorf("Need at least %d mon, %d mds, and %d mgr services in the cluster besides those in node '%s'", o.minMon, o.minMds, o.minMgr, name)
	}
	fmt.Printf("Remaining mon (%d), mds (%d), and mgr (%d) services in the cluster are enough after '%s' enters maintenance mode\n", remains["mon"], remains["mds"], remains["mgr"], name)

	return nil
}

func (o *isNonOsdServiceEnoughOps) DryRun(name string) string {
	return fmt.Sprintf("Will check if there are at least %d mon, %d mds, and %d mgr services in the cluster besides those in node '%s'", o.minMon, o.minMds, o.minMgr, name)
}

type isOkayToStopOsdOps struct {
	cephClient    client.ClientInterface
	clusterClient *microCli.Client
}

func (o *isOkayToStopOsdOps) Run(name string) error {
	disks, err := o.cephClient.GetDisks(o.clusterClient)
	if err != nil {
		return fmt.Errorf("Error getting disks: %v", err)
	}

	okayToStopOSDs := []int64{}
	failedToStopOSDs := []int64{}
	for _, disk := range disks {
		if disk.Location == name {
			if safetyCheckStop(disk.OSD) == nil {
				okayToStopOSDs = append(okayToStopOSDs, disk.OSD)
			} else {
				failedToStopOSDs = append(failedToStopOSDs, disk.OSD)
			}
		}
	}
	fmt.Printf("osd.%v can be safely stopped.\n", okayToStopOSDs)

	if len(failedToStopOSDs) > 0 {
		return fmt.Errorf("osd.%v cannot be safely stopped", failedToStopOSDs)
	}

	return nil
}

func (o *isOkayToStopOsdOps) DryRun(name string) string {
	return fmt.Sprintf("Will check if osds in node '%s' are okay-to-stop.", name)
}

type downgradeFailureDomainOps struct {
	cephClient    client.ClientInterface
	clusterClient *microCli.Client

	confirm bool
}

func (o *downgradeFailureDomainOps) Run(name string) error {
	currentCrushRuleID, err := getDefaultCrushRule()
	if err != nil {
		return err
	}
	hostCrushRuleID, err := getCrushRuleID("microceph_auto_host")
	if err != nil {
		return err
	}
	if currentCrushRuleID != hostCrushRuleID {
		fmt.Println("No need to downgrade failure domain because it's already at 'osd' level or it's using a custom crush rule.")
		return nil
	}

	clusterMembers, err := o.cephClient.GetClusterMembers(o.clusterClient)
	if err != nil {
		return err
	}
	if len(clusterMembers) <= 3 {
		if !o.confirm {
			return fmt.Errorf("Downgrade failure domain is potentially dangerous, please confirm to continue.")
		}
		err = switchFailureDomain("host", "osd")
		if err != nil {
			return err
		}
		fmt.Println("Downgraded failure domain from 'host' to 'osd'.")
	}
	fmt.Println("No need to downgrade failure domain.")
	return nil
}

func (o *downgradeFailureDomainOps) DryRun(name string) string {
	return fmt.Sprintf("Will downgrade failure domain in node '%s' if it's required and confirmed by the operator.", name)
}

type osdSetNooutOps struct {
	confirm bool
}

func (o *osdSetNooutOps) Run(name string) error {
	if !o.confirm {
		fmt.Println("Not setting noout because it's not confirmed.")
		return nil
	}
	err := osdSetNoout()
	if err != nil {
		return err
	}
	fmt.Println("Set osd noout.")
	return nil
}

func (o *osdSetNooutOps) DryRun(name string) string {
	return fmt.Sprintf("Will run `ceph osd set noout` in node '%s' if it's confirmed.", name)
}

type stopOsdOps struct {
	cephClient    client.ClientInterface
	clusterClient *microCli.Client

	killOsd bool
}

func (o *stopOsdOps) Run(name string) error {
	disks, err := o.cephClient.GetDisks(o.clusterClient)
	if err != nil {
		return err
	}
	for _, disk := range disks {
		if disk.Location == name {
			if safetyCheckStop(disk.OSD) != nil {
				return fmt.Errorf("osd.%d is not safe to stop", disk.OSD)
			}

			err = outDownOSD(disk.OSD)
			if err != nil {
				return err
			}
			fmt.Printf("Took osd.%d out and down.\n", disk.OSD)

			if o.killOsd {
				killOSD(disk.OSD)
				fmt.Printf("Killed osd.%d service.\n", disk.OSD)
			} else {
				fmt.Printf("Not killing osd.%d service because it's not confirmed.\n", disk.OSD)
			}
		}
	}
	return nil
}

func (o *stopOsdOps) DryRun(name string) string {
	return fmt.Sprintf("Will stop osds in node '%s' and optionally kill osd services immediately if confirmed.", name)
}

type stopNonOsdOps struct {
	cephClient    client.ClientInterface
	clusterClient *microCli.Client
}

func (o *stopNonOsdOps) Run(name string) error {
	services, err := o.cephClient.GetServices(o.clusterClient)
	if err != nil {
		return err
	}
	for _, service := range services {
		if service.Location == name {
			err = client.SendStopServiceRequestReq(context.Background(), o.clusterClient, name, service.Service)
			if err != nil {
				return err
			}
			fmt.Printf("Stopped %s service.\n", service.Service)
		}
	}
	return nil
}

func (o *stopNonOsdOps) DryRun(name string) string {
	return fmt.Sprintf("Will stop non-osd services in node '%s'.", name)
}

type startNonOsdOps struct {
	cephClient    client.ClientInterface
	clusterClient *microCli.Client
}

func (o *startNonOsdOps) Run(name string) error {
	services, err := o.cephClient.GetServices(o.clusterClient)
	if err != nil {
		return nil
	}
	for _, service := range services {
		if service.Location == name {
			err = snapStart(service.Service, true)
			if err != nil {
				return err
			}
			fmt.Printf("Started %s service.\n", service.Service)
		}
	}
	return nil
}

func (o *startNonOsdOps) DryRun(name string) string {
	return fmt.Sprintf("Will start non-osd services in node '%s'.", name)
}

type startOsdOps struct {
	cephClient    client.ClientInterface
	clusterClient *microCli.Client
}

func (o *startOsdOps) Run(name string) error {
	disks, err := o.cephClient.GetDisks(o.clusterClient)
	if err != nil {
		return err
	}
	for _, disk := range disks {
		if disk.Location == name {
			// bring the OSD in
			err = inOSD(disk.OSD)
			if err != nil {
				return err
			}
			fmt.Printf("Brought osd.%d back in.\n", disk.OSD)
		}
	}
	return nil
}

func (o *startOsdOps) DryRun(name string) string {
	return fmt.Sprintf("Will start osd services in node '%s'.", name)
}

type osdUnsetNooutOps struct {
	confirm bool
}

func (o *osdUnsetNooutOps) Run(name string) error {
	if !o.confirm {
		fmt.Println("Not unsetting noout because it's not confirmed.")
		return nil
	}
	err := osdUnsetNoout()
	if err != nil {
		return err
	}
	fmt.Println("Unset osd noout.")
	return nil
}

func (o *osdUnsetNooutOps) DryRun(name string) string {
	return fmt.Sprintf("Will run `ceph osd unset noout` in node '%s' if it's confirmed.", name)
}

type upgradeFailureDomainOps struct {
	cephClient    client.ClientInterface
	clusterClient *microCli.Client

	confirm bool
}

func (o *upgradeFailureDomainOps) Run(name string) error {
	currentCrushRuleID, err := getDefaultCrushRule()
	if err != nil {
		return err
	}
	osdCrushRuleID, err := getCrushRuleID("microceph_auto_osd")
	if err != nil {
		return err
	}
	if currentCrushRuleID != osdCrushRuleID {
		fmt.Println("No need to upgrade failure domain because it's already at 'host' level or it's using a custom crush rule.")
		return nil
	}

	clusterMembers, err := o.cephClient.GetClusterMembers(o.clusterClient)
	if err != nil {
		return fmt.Errorf("Error getting cluster members: %v", err)
	}
	if len(clusterMembers) >= 3 {
		if !o.confirm {
			return fmt.Errorf("Upgrade failure domain is potentially dangerous, please confirm to continue.")
		}
		err := switchFailureDomain("osd", "host")
		if err != nil {
			return err
		}
		fmt.Println("Upgraded failure domain from 'osd' to 'host'.")
	}
	return nil
}

func (o *upgradeFailureDomainOps) DryRun(name string) string {
	return fmt.Sprintf("Will upgrade failure domain in node '%s' if it's required and confirmed by the operator.", name)
}
