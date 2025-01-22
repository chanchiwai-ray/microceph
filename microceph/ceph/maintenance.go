package ceph

type Maintenance struct {
	Node       string
	ClusterOps ClusterOps
}

func (m *Maintenance) Exit(dryRun, checkOnly bool) []Result {
	results := []Result{}

	// Preflight checks for Exit is currently empty
	preflightChecks := []Operation{}
	results = append(results, RunOperations(m.Node, preflightChecks, dryRun, false)...)

	// Return now if check only
	if checkOnly {
		return results
	}
	// Return now if there's error in preflight checks
	for _, result := range results {
		if result.Error != "" {
			return results
		}
	}

	// Otherwise, continue the rest of the operations

	// idempotently unset noout and start osd service
	operations := []Operation{
		&UnsetNooutOps{ClusterOps: m.ClusterOps},
		&AssertNooutFlagUnsetOps{ClusterOps: m.ClusterOps},
		&StartOsdOps{ClusterOps: m.ClusterOps},
	}

	results = append(results, RunOperations(m.Node, operations, dryRun, false)...)
	return results
}

func (m *Maintenance) Enter(force, dryRun, setNoout, stopOsds, checkOnly bool) []Result {
	results := []Result{}

	preflightChecks := []Operation{
		&CheckOsdOkToStopOps{ClusterOps: m.ClusterOps},
		&CheckNonOsdSvcEnoughOps{ClusterOps: m.ClusterOps, MinMon: 3, MinMds: 1, MinMgr: 1},
	}
	results = append(results, RunOperations(m.Node, preflightChecks, dryRun, false)...)

	// Return now if check only
	if checkOnly {
		return results
	}
	// Return now if there's error in preflight checks and if it's not forced
	for _, result := range results {
		if result.Error != "" && !force {
			return results
		}
	}

	// Otherwise, continue the rest of the operations

	operations := []Operation{}
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

	results = append(results, RunOperations(m.Node, operations, dryRun, force)...)
	return results
}
