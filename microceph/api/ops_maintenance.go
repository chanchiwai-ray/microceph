package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/canonical/lxd/lxd/response"
	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microceph/microceph/api/types"
	"github.com/canonical/microceph/microceph/ceph"
	"github.com/canonical/microcluster/v2/rest"
	"github.com/canonical/microcluster/v2/state"
)

// /ops/maintenance endpoint.
var opsMaintenanceCmd = rest.Endpoint{
	Path: "ops/maintenance/",
	Put:  rest.EndpointAction{Handler: cmdPutMaintenance, ProxyTarget: false},
}

// cmdPutMaintenance bring a node in or out of maintenance
func cmdPutMaintenance(s state.State, r *http.Request) response.Response {
	var maintenancePut types.MaintenancePut
	var plan types.MaintenancePlan

	err := json.NewDecoder(r.Body).Decode(&maintenancePut)
	if err != nil {
		logger.Errorf("failed decoding body: %v", err)
		return response.InternalError(err)
	}

	node := s.Name()
	maintenance := ceph.Maintenance{
		Node: node,
		ClusterOps: ceph.ClusterOps{
			State:   s,
			Context: r.Context(),
		},
	}

	status := maintenancePut.Status
	switch status {
	case "maintenance":
		plan, err = maintenance.Enter(maintenancePut.Force, maintenancePut.DryRun, maintenancePut.SetNoout, maintenancePut.StopOsds)
	case "non-maintenance":
		plan, err = maintenance.Exit(maintenancePut.DryRun)
	default:
		err = fmt.Errorf("unknown status encounter: '%s', can only be 'maintenance' or 'non-maintenance'", status)
	}

	if err != nil {
		logger.Errorf("failed bring the node (%s) in or out of maintenance: %v", node, err)
		return response.SyncResponse(false, err)
	}

	return response.SyncResponse(true, plan)
}
