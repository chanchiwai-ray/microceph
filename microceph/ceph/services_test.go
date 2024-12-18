package ceph

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/canonical/lxd/shared/api"
	"github.com/canonical/microceph/microceph/api/types"
	"github.com/canonical/microceph/microceph/tests"

	"github.com/canonical/microceph/microceph/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type servicesSuite struct {
	tests.BaseSuite
	TestStateInterface *mocks.StateInterface
}

func TestServices(t *testing.T) {
	suite.Run(t, new(servicesSuite))
}

// Set up test suite
func (s *servicesSuite) SetupTest() {
	s.BaseSuite.SetupTest()

	s.TestStateInterface = mocks.NewStateInterface(s.T())
	u := api.NewURL()
	state := mocks.MockState{
		URL:         u,
		ClusterName: "foohost",
	}
	s.TestStateInterface.On("ClusterState").Return(state).Maybe()
}

func addOsdDumpExpectations(r *mocks.Runner) {
	osdDumpObj := "{\"osds\":[{\"up\":1,\"uuid\":\"bfbbd27a-472f-4771-a6f7-7c5db9803d41\"}]}"
	osdDump, _ := json.Marshal(osdDumpObj)

	// Expect osd service worker query
	r.On("RunCommand", []interface{}{
		"ceph", "osd", "dump", "-f", "json-pretty",
	}...).Return(string(osdDump[:]), nil).Twice()
}

func addMonDumpExpectations(r *mocks.Runner) {
	monDumpObj := "{\"mons\":[{\"name\":\"bfbbd27a\"}]}"
	monDump, _ := json.Marshal(monDumpObj)

	// Expect mon service worker query
	r.On("RunCommand", []interface{}{
		"ceph", "mon", "dump", "-f", "json-pretty",
	}...).Return(string(monDump[:]), nil).Twice()
}

func addServiceStopExpectations(r *mocks.Runner, services []string) {
	for _, service := range services {
		r.On("RunCommand", []interface{}{
			"snapctl", "stop", fmt.Sprintf("microceph.%s", service),
		}...).Return("ok", nil).Once()
	}
}

func addServiceStopErrorExpectations(r *mocks.Runner, services []string) {
	for _, service := range services {
		r.On("RunCommand", []interface{}{
			"snapctl", "stop", fmt.Sprintf("microceph.%s", service),
		}...).Return("fail", fmt.Errorf("some reasons")).Once()
	}
}

func addServiceStartExpectations(r *mocks.Runner, services []string) {
	for _, service := range services {
		r.On("RunCommand", []interface{}{
			"snapctl", "start", fmt.Sprintf("microceph.%s", service),
		}...).Return("ok", nil).Once()
	}
}

func addServiceStartErrorExpectations(r *mocks.Runner, services []string) {
	for _, service := range services {
		r.On("RunCommand", []interface{}{
			"snapctl", "start", fmt.Sprintf("microceph.%s", service),
		}...).Return("fail", fmt.Errorf("some reasons")).Once()
	}
}

func addServiceRestartExpectations(r *mocks.Runner, services []string) {
	for _, service := range services {
		r.On("RunCommand", []interface{}{
			"snapctl", "restart", fmt.Sprintf("microceph.%s", service),
		}...).Return("ok", nil).Once()
	}
}

func (s *servicesSuite) TestStopService() {
	// any service that returns from ListServices is okay
	services := types.Services{
		types.Service{Service: "mon", Location: "foohost"},
	}

	r := mocks.NewRunner()
	addServiceStopExpectations(r, services)
	// patch processExec singleton
	processExec = r

	// services are defined
	for _, svc := range services {
		err := StopService(services, svc, "foohost")
		assert.NoError(s.T(), err)
	}

	// unknown service will be skipped
	err := StopService(services, "unknown", "foohost")
	assert.NoError(s.T(), err)
}

func (s *servicesSuite) TestStopServiceError() {
	// any service that returns from ListServices is okay
	services := types.Services{
		types.Service{Service: "mon", Location: "foohost"},
	}

	r := mocks.NewRunner()
	addServiceStopErrorExpectations(r, services)
	// patch processExec singleton
	processExec = r

	// services are defined but command failed
	for _, svc := range services {
		err := StopService(services, svc, "foohost")
		assert.Error(s.T(), err)
	}
}

func (s *servicesSuite) TestStartService() {
	// any service that returns from ListServices is okay
	services := types.Services{
		types.Service{Service: "mon", Location: "foohost"},
	}

	r := mocks.NewRunner()
	addServiceStartExpectations(r, services)
	// patch processExec singleton
	processExec = r

	// services are defined
	for _, svc := range services {
		err := StartService(services, svc, "foohost")
		assert.NoError(s.T(), err)
	}

	// unknown service will be skipped
	err := StartService(services, "unknown", "foohost")
	assert.NoError(s.T(), err)
}

func (s *servicesSuite) TestStartServiceError() {
	// any service that returns from ListServices is okay
	services := types.Services{
		types.Service{Service: "mon", Location: "foohost"},
	}

	r := mocks.NewRunner()
	addServiceStartErrorExpectations(r, services)
	// patch processExec singleton
	processExec = r

	// services are defined but command failed
	for _, svc := range services {
		err := StartService(services, svc, "foohost")
		assert.Error(s.T(), err)
	}
}

func (s *servicesSuite) TestRestartInvalidService() {
	services := types.Services{}
	err := RestartCephService(services, "InvalidService", "foohost")
	assert.ErrorContains(s.T(), err, "no handler defined")
}

func (s *servicesSuite) TestRestartServiceWorkerSuccess() {
	ts := []string{"mon", "osd"} // test services

	r := mocks.NewRunner(s.T())
	addMonDumpExpectations(r)
	addOsdDumpExpectations(r)
	addServiceRestartExpectations(r, ts)
	processExec = r

	services := types.Services{
		types.Service{Service: "mon", Location: "foohost"},
		types.Service{Service: "osd", Location: "foohost"},
	}

	// Handler is defined for both mon and osd services.
	err := RestartCephService(services, "mon", "foohost")
	assert.NoError(s.T(), err)

	err = RestartCephService(services, "osd", "foohost")
	assert.NoError(s.T(), err)
}

// TestCleanService tests the cleanService function.
func (s *servicesSuite) TestCleanService() {
	s.CopyCephConfigs()
	svcPath := filepath.Join(s.Tmp, "SNAP_COMMON", "data", "mon", "ceph-foo-host")
	os.MkdirAll(svcPath, 0770)
	cleanService("foo-host", "mon")
	assert.NoDirExists(s.T(), svcPath)
}
