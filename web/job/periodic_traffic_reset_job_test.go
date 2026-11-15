package job

import (
	"errors"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
)

func TestPeriodicTrafficResetSynchronizesEverySuccessfulOperation(t *testing.T) {
	resetError := errors.New("forced reset failure")
	for _, test := range []struct {
		name                  string
		inboundErr, clientErr error
		running, wantRestart  bool
	}{
		{"both succeed", nil, nil, true, true},
		{"only inbound succeeds", nil, resetError, true, true},
		{"only clients succeed", resetError, nil, true, true},
		{"neither succeeds", resetError, resetError, true, false},
		{"manual stop", nil, nil, false, false},
		{"manual stop and inbound succeeds", nil, resetError, false, false},
		{"manual stop and clients succeed", resetError, nil, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			inbounds := &fakePeriodicTrafficResetInboundService{
				inbounds:   []*model.Inbound{{Id: 7}, {Id: 9}},
				inboundErr: test.inboundErr, clientErr: test.clientErr,
			}
			xray := &fakePeriodicTrafficResetXrayService{running: test.running}
			job := &PeriodicTrafficResetJob{period: "daily", inboundService: inbounds, xrayService: xray}
			job.Run()
			if inbounds.period != "daily" || len(inbounds.inboundIDs) != 2 || len(inbounds.clientIDs) != 2 || inbounds.inboundIDs[0] != 7 || inbounds.inboundIDs[1] != 9 || inbounds.clientIDs[0] != 7 || inbounds.clientIDs[1] != 9 {
				t.Fatalf("reset did not attempt every operation: %+v", inbounds)
			}
			wantRestarts := 0
			if test.wantRestart {
				wantRestarts = 1
			}
			if xray.restartCount != wantRestarts {
				t.Fatalf("restart requests = %d, want %d", xray.restartCount, wantRestarts)
			}
		})
	}
}

func TestPeriodicTrafficResetSkipsRuntimeWithoutMatchingInbounds(t *testing.T) {
	for _, queryError := range []error{nil, errors.New("forced query failure")} {
		inbounds := &fakePeriodicTrafficResetInboundService{queryErr: queryError}
		xray := &fakePeriodicTrafficResetXrayService{running: true}
		job := &PeriodicTrafficResetJob{period: "daily", inboundService: inbounds, xrayService: xray}
		job.Run()
		if len(inbounds.inboundIDs) != 0 || len(inbounds.clientIDs) != 0 || xray.restartCount != 0 {
			t.Fatalf("empty or failed lookup attempted a reset: %+v, %+v", inbounds, xray)
		}
	}
}

type fakePeriodicTrafficResetInboundService struct {
	inbounds                        []*model.Inbound
	period                          string
	inboundIDs, clientIDs           []int
	queryErr, inboundErr, clientErr error
}

func (s *fakePeriodicTrafficResetInboundService) GetInboundsByTrafficReset(period string) ([]*model.Inbound, error) {
	s.period = period
	return s.inbounds, s.queryErr
}

func (s *fakePeriodicTrafficResetInboundService) ResetInboundTraffic(id int) error {
	s.inboundIDs = append(s.inboundIDs, id)
	return s.inboundErr
}

func (s *fakePeriodicTrafficResetInboundService) ResetAllClientTraffics(id int) error {
	s.clientIDs = append(s.clientIDs, id)
	return s.clientErr
}

type fakePeriodicTrafficResetXrayService struct {
	running      bool
	restartCount int
}

func (s *fakePeriodicTrafficResetXrayService) IsXrayRunning() bool {
	return s.running
}

func (s *fakePeriodicTrafficResetXrayService) SetToNeedRestart() {
	s.restartCount++
}
