// Package client provides a full Go API client.
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/canonical/lxd/shared/api"
	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/v2/client"

	"github.com/canonical/microceph/microceph/api/types"
)

// ExitMaintenance sends the request to '/ops/maintenance/{node}' endpoint to bring a node out of
// maintenance mode.
func ExitMaintenance(ctx context.Context, c *client.Client, node string, dryRun bool) (types.MaintenancePlan, error) {
	queryCtx, cancel := context.WithTimeout(ctx, time.Second*120)
	defer cancel()

	var plan types.MaintenancePlan
	data := types.MaintenancePut{
		Status:           "non-maintenance",
		MaintenanceFlags: types.MaintenanceFlags{DryRun: dryRun},
	}

	c = c.UseTarget(node)
	err := c.Query(queryCtx, "PUT", types.ExtendedPathPrefix, api.NewURL().Path("ops", "maintenance"), data, &plan)
	if err != nil {
		url := c.URL()
		logger.Errorf("error bringing node '%s' into maintenance: %v", node, err)
		return plan, fmt.Errorf("failed Forwarding To: %s: %w", url.String(), err)
	}
	return plan, nil
}

// EnterMaintenance sends the request to '/ops/maintenance/' endpoint to bring a node into
// maintenance mode.
func EnterMaintenance(ctx context.Context, c *client.Client, node string, force, dryRun, setNoout, stopOsds bool) (types.MaintenancePlan, error) {
	queryCtx, cancel := context.WithTimeout(ctx, time.Second*120)
	defer cancel()

	var plan types.MaintenancePlan
	data := types.MaintenancePut{
		Status:                "maintenance",
		MaintenanceFlags:      types.MaintenanceFlags{DryRun: dryRun},
		MaintenanceEnterFlags: types.MaintenanceEnterFlags{Force: force, SetNoout: setNoout, StopOsds: stopOsds},
	}

	c = c.UseTarget(node)
	err := c.Query(queryCtx, "PUT", types.ExtendedPathPrefix, api.NewURL().Path("ops", "maintenance"), data, &plan)
	if err != nil {
		url := c.URL()
		logger.Errorf("error bringing node '%s' out of maintenance: %v", node, err)
		return plan, fmt.Errorf("failed Forwarding To: %s: %w", url.String(), err)
	}
	return plan, nil
}
