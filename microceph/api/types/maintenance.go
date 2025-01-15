// Package types provides shared types and structs.
package types

type MaintenancePlan []string

// Options for bringing a node into or out of maintenance
type MaintenanceFlags struct {
	DryRun bool `json:"dry_run"`
}

// Options for bringing a node into maintenance
type MaintenanceEnterFlags struct {
	Force    bool `json:"force"`
	SetNoout bool `json:"set_noout"`
	StopOsds bool `json:"stop_osds"`
}

// MaintenancePut holds data structure for bringing a node into or out of maintenance
type MaintenancePut struct {
	Status string `json:"status" yaml:"status"`
	MaintenanceFlags
	MaintenanceEnterFlags
}
