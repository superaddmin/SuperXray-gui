package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/op/go-logging"
	panelcore "github.com/superaddmin/SuperXray-gui/v2/core"
	"github.com/superaddmin/SuperXray-gui/v2/logger"
	"github.com/superaddmin/SuperXray-gui/v2/web/entity"

	"github.com/gin-gonic/gin"
)

func TestCoreControllerListsInstances(t *testing.T) {
	initCoreControllerTestLogger(t)
	router := newCoreControllerTestRouter(&fakeCoreService{
		instances: []panelcore.Instance{
			{ID: "default-xray", CoreType: panelcore.CoreTypeXray, Status: panelcore.Status{State: panelcore.StateStopped}},
			{ID: "experimental-sing-box", CoreType: panelcore.CoreTypeSingBox, Status: panelcore.Status{State: panelcore.StateNotInstalled}},
		},
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/panel/api/cores/instances", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	msg := decodeCoreMsg(t, recorder)
	if !msg.Success {
		t.Fatalf("success = false, msg = %q", msg.Msg)
	}
	instances := msg.Obj.([]any)
	if len(instances) != 2 {
		t.Fatalf("len(instances) = %d, want 2", len(instances))
	}
}

func TestCoreControllerReturnsErrorForUnsupportedLifecycle(t *testing.T) {
	initCoreControllerTestLogger(t)
	router := newCoreControllerTestRouter(&fakeCoreService{
		startResult: panelcore.LifecycleResult{State: panelcore.StateError, ErrorMsg: panelcore.ErrLifecycleUnsupported.Error()},
		startErr:    panelcore.ErrLifecycleUnsupported,
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example.test/panel/api/cores/instances/default-xray/start", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	msg := decodeCoreMsg(t, recorder)
	if msg.Success {
		t.Fatal("expected unsupported lifecycle to return success=false")
	}
}

func initCoreControllerTestLogger(t *testing.T) {
	t.Helper()
	t.Setenv("XUI_LOG_FOLDER", filepath.Join(t.TempDir(), "logs"))
	logger.InitLogger(logging.DEBUG)
	t.Cleanup(logger.CloseLogger)
}

func newCoreControllerTestRouter(service coreAPIService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewCoreController(router.Group("/panel/api/cores"), service)
	return router
}

func decodeCoreMsg(t *testing.T, recorder *httptest.ResponseRecorder) entity.Msg {
	t.Helper()

	var raw struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
		Obj     any    `json:"obj"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return entity.Msg{Success: raw.Success, Msg: raw.Msg, Obj: raw.Obj}
}

type fakeCoreService struct {
	instances   []panelcore.Instance
	instance    panelcore.Instance
	status      panelcore.Status
	result      panelcore.LifecycleResult
	err         error
	startResult panelcore.LifecycleResult
	startErr    error
}

func (s *fakeCoreService) List(context.Context) ([]panelcore.Instance, error) {
	return s.instances, s.err
}

func (s *fakeCoreService) Get(context.Context, string) (panelcore.Instance, error) {
	return s.instance, s.err
}

func (s *fakeCoreService) Status(context.Context, string) (panelcore.Status, error) {
	return s.status, s.err
}

func (s *fakeCoreService) Validate(context.Context, string) (panelcore.LifecycleResult, error) {
	return s.result, s.err
}

func (s *fakeCoreService) Start(context.Context, string) (panelcore.LifecycleResult, error) {
	return s.startResult, s.startErr
}

func (s *fakeCoreService) Stop(context.Context, string) (panelcore.LifecycleResult, error) {
	return s.result, s.err
}

func (s *fakeCoreService) Restart(context.Context, string) (panelcore.LifecycleResult, error) {
	return s.result, s.err
}
