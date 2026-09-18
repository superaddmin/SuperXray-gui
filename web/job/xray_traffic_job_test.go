package job

import (
	"errors"
	"testing"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/xray"
)

func TestXrayTrafficJobAcknowledgesInboundAndOutboundSeparately(t *testing.T) {
	xrayService := &fakeTrafficXrayService{
		running: true,
		batches: []trafficTestBatch{
			{
				traffics: []*xray.Traffic{
					{IsInbound: true, Tag: "inbound-a", Up: 10, Down: 1},
					{IsOutbound: true, Tag: "outbound-a", Up: 20, Down: 2},
				},
				clients: []*xray.ClientTraffic{{Email: "client-a@example.test", Up: 5, Down: 1}},
			},
			{
				traffics: []*xray.Traffic{
					{IsInbound: true, Tag: "inbound-a", Up: 3, Down: 4},
					{IsOutbound: true, Tag: "outbound-a", Up: 7, Down: 8},
				},
				clients: []*xray.ClientTraffic{{Email: "client-a@example.test", Up: 2, Down: 3}},
			},
		},
	}
	inboundService := &fakeTrafficInboundService{}
	outboundService := &fakeTrafficOutboundService{
		results: []trafficAddResult{
			{err: errors.New("temporary outbound database failure")},
			{},
		},
	}
	job := newTrafficTestJob(xrayService, inboundService, outboundService)

	job.Run()
	job.Run()

	if len(inboundService.calls) != 2 {
		t.Fatalf("inbound AddTraffic call count = %d, want 2", len(inboundService.calls))
	}
	assertTrafficDelta(t, inboundService.calls[0].traffics, "inbound-a", 10, 1)
	assertClientTrafficDelta(t, inboundService.calls[0].clients, "client-a@example.test", 5, 1)
	assertTrafficDelta(t, inboundService.calls[1].traffics, "inbound-a", 3, 4)
	assertClientTrafficDelta(t, inboundService.calls[1].clients, "client-a@example.test", 2, 3)

	if len(outboundService.calls) != 2 {
		t.Fatalf("outbound AddTraffic call count = %d, want 2", len(outboundService.calls))
	}
	assertTrafficDelta(t, outboundService.calls[0].traffics, "outbound-a", 20, 2)
	assertTrafficDelta(t, outboundService.calls[1].traffics, "outbound-a", 27, 10)
	if !job.pending.empty() {
		t.Fatalf("pending traffic after successful retry = %+v, want empty", job.pending)
	}
}

func TestXrayTrafficJobRetriesPendingTrafficWhileXrayIsStopped(t *testing.T) {
	xrayService := &fakeTrafficXrayService{
		running: true,
		batches: []trafficTestBatch{{
			traffics: []*xray.Traffic{{IsInbound: true, Tag: "offline-inbound", Up: 9, Down: 4}},
			clients:  []*xray.ClientTraffic{{Email: "offline@example.test", Up: 3, Down: 2}},
		}},
	}
	inboundService := &fakeTrafficInboundService{
		results: []trafficAddResult{
			{err: errors.New("temporary inbound database failure"), needRestart: true},
			{needRestart: true},
		},
	}
	outboundService := &fakeTrafficOutboundService{}
	job := newTrafficTestJob(xrayService, inboundService, outboundService)

	job.Run()
	if xrayService.restartCalls != 1 {
		t.Fatalf("SetToNeedRestart call count = %d, want 1 after failed runtime-affecting write", xrayService.restartCalls)
	}
	xrayService.running = false
	job.Run()

	if xrayService.restartCalls != 1 {
		t.Fatalf("SetToNeedRestart call count = %d, want offline retry not to schedule another restart", xrayService.restartCalls)
	}
	if xrayService.getCalls != 1 {
		t.Fatalf("GetXrayTraffic call count = %d, want 1 while second run is offline", xrayService.getCalls)
	}
	if len(inboundService.calls) != 2 {
		t.Fatalf("inbound AddTraffic call count = %d, want offline retry", len(inboundService.calls))
	}
	assertTrafficDelta(t, inboundService.calls[1].traffics, "offline-inbound", 9, 4)
	assertClientTrafficDelta(t, inboundService.calls[1].clients, "offline@example.test", 3, 2)
	if !job.pending.empty() {
		t.Fatalf("pending traffic after offline retry = %+v, want empty", job.pending)
	}
}

func TestXrayTrafficJobRetriesPendingTrafficWhenCollectionFails(t *testing.T) {
	xrayService := &fakeTrafficXrayService{
		running: true,
		batches: []trafficTestBatch{
			{traffics: []*xray.Traffic{{IsOutbound: true, Tag: "collection-error", Up: 6, Down: 7}}},
			{err: errors.New("xray stats unavailable")},
		},
	}
	inboundService := &fakeTrafficInboundService{}
	outboundService := &fakeTrafficOutboundService{
		results: []trafficAddResult{
			{err: errors.New("temporary outbound database failure")},
			{},
		},
	}
	job := newTrafficTestJob(xrayService, inboundService, outboundService)

	job.Run()
	job.Run()

	if xrayService.getCalls != 2 {
		t.Fatalf("GetXrayTraffic call count = %d, want 2", xrayService.getCalls)
	}
	if len(outboundService.calls) != 2 {
		t.Fatalf("outbound AddTraffic call count = %d, want retry after collection error", len(outboundService.calls))
	}
	assertTrafficDelta(t, outboundService.calls[1].traffics, "collection-error", 6, 7)
	if !job.pending.empty() {
		t.Fatalf("pending traffic after collection-error retry = %+v, want empty", job.pending)
	}
}

func TestXrayTrafficJobDoesNotScheduleRestartWhenXrayStopsDuringPersistence(t *testing.T) {
	xrayService := &fakeTrafficXrayService{
		running: true,
		batches: []trafficTestBatch{{
			traffics: []*xray.Traffic{{IsInbound: true, Tag: "stopped-during-write", Up: 1}},
		}},
	}
	inboundService := &fakeTrafficInboundService{
		results: []trafficAddResult{{needRestart: true}},
		onAdd: func() {
			xrayService.running = false
		},
	}
	job := newTrafficTestJob(xrayService, inboundService, &fakeTrafficOutboundService{})

	job.Run()

	if xrayService.restartCalls != 0 {
		t.Fatalf("SetToNeedRestart call count = %d, want 0 after Xray stopped during persistence", xrayService.restartCalls)
	}
}

func TestXrayTrafficJobDiscardsInvalidDeltas(t *testing.T) {
	xrayService := &fakeTrafficXrayService{
		running: true,
		batches: []trafficTestBatch{{
			traffics: []*xray.Traffic{
				nil,
				{IsInbound: true, Up: 1},
				{IsInbound: true, Tag: "negative-inbound", Up: -1},
				{IsInbound: true, IsOutbound: true, Tag: "ambiguous", Up: 1},
				{IsInbound: true, Tag: "zero-inbound"},
				{IsInbound: true, Tag: "valid-inbound", Up: 2, Down: 3},
				{IsOutbound: true, Tag: "valid-outbound", Up: 4, Down: 5},
			},
			clients: []*xray.ClientTraffic{
				nil,
				{Up: 1},
				{Email: "negative@example.test", Down: -1},
				{Email: "zero@example.test"},
				{Email: "valid@example.test", Up: 6, Down: 7},
			},
		}},
	}
	inboundService := &fakeTrafficInboundService{}
	outboundService := &fakeTrafficOutboundService{}
	job := newTrafficTestJob(xrayService, inboundService, outboundService)

	job.Run()

	if len(inboundService.calls) != 1 || len(inboundService.calls[0].traffics) != 1 || len(inboundService.calls[0].clients) != 1 {
		t.Fatalf("inbound call = %+v, want one valid inbound and client delta", inboundService.calls)
	}
	assertTrafficDelta(t, inboundService.calls[0].traffics, "valid-inbound", 2, 3)
	assertClientTrafficDelta(t, inboundService.calls[0].clients, "valid@example.test", 6, 7)
	if len(outboundService.calls) != 1 || len(outboundService.calls[0].traffics) != 1 {
		t.Fatalf("outbound call = %+v, want one valid outbound delta", outboundService.calls)
	}
	assertTrafficDelta(t, outboundService.calls[0].traffics, "valid-outbound", 4, 5)
	if !job.pending.empty() {
		t.Fatalf("pending traffic after valid writes = %+v, want empty", job.pending)
	}
}

type trafficTestBatch struct {
	traffics []*xray.Traffic
	clients  []*xray.ClientTraffic
	err      error
}

type trafficAddResult struct {
	err         error
	needRestart bool
}

type trafficAddCall struct {
	traffics []*xray.Traffic
	clients  []*xray.ClientTraffic
}

type fakeTrafficSettingService struct{}

func (s *fakeTrafficSettingService) GetExternalTrafficInformEnable() (bool, error) {
	return false, nil
}

func (s *fakeTrafficSettingService) GetExternalTrafficInformURI() (string, error) {
	return "", nil
}

type fakeTrafficXrayService struct {
	running      bool
	batches      []trafficTestBatch
	getCalls     int
	restartCalls int
}

func (s *fakeTrafficXrayService) IsXrayRunning() bool {
	return s.running
}

func (s *fakeTrafficXrayService) GetXrayTraffic() ([]*xray.Traffic, []*xray.ClientTraffic, error) {
	s.getCalls++
	if s.getCalls > len(s.batches) {
		return nil, nil, errors.New("unexpected GetXrayTraffic call")
	}
	batch := s.batches[s.getCalls-1]
	return batch.traffics, batch.clients, batch.err
}

func (s *fakeTrafficXrayService) SetToNeedRestart() {
	s.restartCalls++
}

type fakeTrafficInboundService struct {
	results []trafficAddResult
	calls   []trafficAddCall
	onAdd   func()
}

func (s *fakeTrafficInboundService) AddTraffic(traffics []*xray.Traffic, clients []*xray.ClientTraffic) (error, bool) {
	s.calls = append(s.calls, copyTrafficAddCall(traffics, clients))
	if s.onAdd != nil {
		s.onAdd()
	}
	return nextTrafficAddResult(&s.results)
}

func (s *fakeTrafficInboundService) GetOnlineClients() []string {
	return nil
}

func (s *fakeTrafficInboundService) GetClientsLastOnline() (map[string]int64, error) {
	return nil, nil
}

func (s *fakeTrafficInboundService) GetAllInbounds() ([]*model.Inbound, error) {
	return nil, nil
}

type fakeTrafficOutboundService struct {
	results []trafficAddResult
	calls   []trafficAddCall
}

func (s *fakeTrafficOutboundService) AddTraffic(traffics []*xray.Traffic, clients []*xray.ClientTraffic) (error, bool) {
	s.calls = append(s.calls, copyTrafficAddCall(traffics, clients))
	return nextTrafficAddResult(&s.results)
}

func (s *fakeTrafficOutboundService) GetOutboundsTraffic() ([]*model.OutboundTraffics, error) {
	return nil, nil
}

func newTrafficTestJob(
	xrayService trafficXrayService,
	inboundService trafficInboundService,
	outboundService trafficOutboundService,
) *XrayTrafficJob {
	return &XrayTrafficJob{
		settingService:  &fakeTrafficSettingService{},
		xrayService:     xrayService,
		inboundService:  inboundService,
		outboundService: outboundService,
	}
}

func nextTrafficAddResult(results *[]trafficAddResult) (error, bool) {
	if len(*results) == 0 {
		return nil, false
	}
	result := (*results)[0]
	*results = (*results)[1:]
	return result.err, result.needRestart
}

func copyTrafficAddCall(traffics []*xray.Traffic, clients []*xray.ClientTraffic) trafficAddCall {
	call := trafficAddCall{
		traffics: make([]*xray.Traffic, 0, len(traffics)),
		clients:  make([]*xray.ClientTraffic, 0, len(clients)),
	}
	for _, traffic := range traffics {
		copy := *traffic
		call.traffics = append(call.traffics, &copy)
	}
	for _, client := range clients {
		copy := *client
		call.clients = append(call.clients, &copy)
	}
	return call
}

func assertTrafficDelta(t *testing.T, traffics []*xray.Traffic, tag string, wantUp, wantDown int64) {
	t.Helper()
	for _, traffic := range traffics {
		if traffic.Tag == tag {
			if traffic.Up != wantUp || traffic.Down != wantDown {
				t.Fatalf("traffic %q = %d/%d, want %d/%d", tag, traffic.Up, traffic.Down, wantUp, wantDown)
			}
			return
		}
	}
	t.Fatalf("traffic %q not found in %+v", tag, traffics)
}

func assertClientTrafficDelta(t *testing.T, traffics []*xray.ClientTraffic, email string, wantUp, wantDown int64) {
	t.Helper()
	for _, traffic := range traffics {
		if traffic.Email == email {
			if traffic.Up != wantUp || traffic.Down != wantDown {
				t.Fatalf("client traffic %q = %d/%d, want %d/%d", email, traffic.Up, traffic.Down, wantUp, wantDown)
			}
			return
		}
	}
	t.Fatalf("client traffic %q not found in %+v", email, traffics)
}
