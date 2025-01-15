package ceph

import (
	"fmt"
)

type Maintenance struct {
	Node       string
	ClusterOps ClusterOps
}

func (m *Maintenance) Exit(dryRun bool) ([]string, error) {
	// idempotently unset noout and start osd service
	operations := []Operation{
		&UnsetNooutOps{ClusterOps: m.ClusterOps},
		&AssertNooutFlagUnsetOps{ClusterOps: m.ClusterOps},
		&StartOsdOps{ClusterOps: m.ClusterOps},
	}

	plan, err := RunOperations(m.Node, operations, dryRun, false)
	if err != nil {
		return []string{}, fmt.Errorf("failed to exit maintenance mode: %v", err)
	}

	return plan, nil
}

func (m *Maintenance) Enter(force, dryRun, setNoout, stopOsds bool) ([]string, error) {
	operations := []Operation{
		&CheckOsdOkToStopOps{ClusterOps: m.ClusterOps},
		&CheckNonOsdSvcEnoughOps{ClusterOps: m.ClusterOps, MinMon: 3, MinMds: 1, MinMgr: 1},
	}

	// optionally set noout
	if setNoout {
		operations = append(operations, []Operation{
			&SetNooutOps{ClusterOps: m.ClusterOps},
			&AssertNooutFlagSetOps{ClusterOps: m.ClusterOps},
		}...)
	}

	// optionally stop osd service
	if stopOsds {
		operations = append(operations, []Operation{
			&StopOsdOps{ClusterOps: m.ClusterOps},
		}...)
	}

	plan, err := RunOperations(m.Node, operations, dryRun, force)
	if err != nil {
		return []string{}, fmt.Errorf("failed to enter maintenance mode: %v", err)
	}

	return plan, nil
}
