package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/canonical/lxd/lxd/response"
	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microceph/microceph/api/types"
	"github.com/canonical/microceph/microceph/ceph"
	"github.com/canonical/microcluster/v2/rest"
	"github.com/canonical/microcluster/v2/state"
	"github.com/gorilla/mux"
)

// /ops/maintenance/{node} endpoint.
var opsMaintenanceNodeCmd = rest.Endpoint{
	Path: "ops/maintenance/{node}",
	Put:  rest.EndpointAction{Handler: cmdPutMaintenance, ProxyTarget: true},
}

// cmdPutMaintenance bring a node in or out of maintenance
func cmdPutMaintenance(s state.State, r *http.Request) response.Response {
	var results []ceph.Result
	var maintenancePut types.MaintenancePut

	node, err := url.PathUnescape(mux.Vars(r)["node"])
	if err != nil {
		return response.BadRequest(err)
	}

	err = json.NewDecoder(r.Body).Decode(&maintenancePut)
	if err != nil {
		logger.Errorf("failed decoding body: %v", err)
		return response.InternalError(err)
	}

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
		results = maintenance.Enter(maintenancePut.Force, maintenancePut.DryRun, maintenancePut.SetNoout, maintenancePut.StopOsds, maintenancePut.CheckOnly)
	case "non-maintenance":
		results = maintenance.Exit(maintenancePut.DryRun, maintenancePut.CheckOnly)
	default:
		err = fmt.Errorf("unknown status encounter: '%s', can only be 'maintenance' or 'non-maintenance'", status)
		return response.BadRequest(err)
	}

	for _, result := range results {
		if result.Error != "" && !maintenancePut.Force {
			return response.SyncResponse(false, results)
		}
	}
	return response.SyncResponse(true, results)
}
