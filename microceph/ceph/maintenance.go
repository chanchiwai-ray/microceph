package ceph

type Maintenance struct {
	Node       string
	ClusterOps ClusterOps
}

func (m *Maintenance) Exit(dryRun bool) []Result {
	// idempotently unset noout and start osd service
	operations := []Operation{
		&UnsetNooutOps{ClusterOps: m.ClusterOps},
		&AssertNooutFlagUnsetOps{ClusterOps: m.ClusterOps},
		&StartOsdOps{ClusterOps: m.ClusterOps},
	}

	return RunOperations(m.Node, operations, dryRun, false)
}

func (m *Maintenance) Enter(force, dryRun, setNoout, stopOsds bool) []Result {
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

	return RunOperations(m.Node, operations, dryRun, force)
}
