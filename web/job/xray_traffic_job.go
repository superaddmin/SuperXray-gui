package job

import (
	"encoding/json"
	"sync"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/logger"
	"github.com/superaddmin/SuperXray-gui/v2/web/service"
	"github.com/superaddmin/SuperXray-gui/v2/web/websocket"
	"github.com/superaddmin/SuperXray-gui/v2/xray"

	"github.com/valyala/fasthttp"
)

type trafficSettingService interface {
	GetExternalTrafficInformEnable() (bool, error)
	GetExternalTrafficInformURI() (string, error)
}

type trafficXrayService interface {
	IsXrayRunning() bool
	GetXrayTraffic() ([]*xray.Traffic, []*xray.ClientTraffic, error)
	SetToNeedRestart()
}

type trafficInboundService interface {
	AddTraffic([]*xray.Traffic, []*xray.ClientTraffic) (error, bool)
	GetOnlineClients() []string
	GetClientsLastOnline() (map[string]int64, error)
	GetAllInbounds() ([]*model.Inbound, error)
}

type trafficOutboundService interface {
	AddTraffic([]*xray.Traffic, []*xray.ClientTraffic) (error, bool)
	GetOutboundsTraffic() ([]*model.OutboundTraffics, error)
}

type trafficDelta struct {
	up   int64
	down int64
}

type pendingTraffic struct {
	inbounds  map[string]trafficDelta
	clients   map[string]trafficDelta
	outbounds map[string]trafficDelta
}

// XrayTrafficJob collects and processes traffic statistics from Xray, updating the database and optionally informing external APIs.
type XrayTrafficJob struct {
	mu              sync.Mutex
	settingService  trafficSettingService
	xrayService     trafficXrayService
	inboundService  trafficInboundService
	outboundService trafficOutboundService
	pending         pendingTraffic
}

// NewXrayTrafficJob creates a new traffic collection job instance.
func NewXrayTrafficJob() *XrayTrafficJob {
	return &XrayTrafficJob{
		settingService:  &service.SettingService{},
		xrayService:     &service.XrayService{},
		inboundService:  &service.InboundService{},
		outboundService: &service.OutboundService{},
	}
}

// Run collects traffic statistics from Xray and updates the database, triggering restart if needed.
func (j *XrayTrafficJob) Run() {
	j.mu.Lock()
	defer j.mu.Unlock()

	var traffics []*xray.Traffic
	var clientTraffics []*xray.ClientTraffic
	collected := false
	xrayRunning := j.xrayService.IsXrayRunning()
	if xrayRunning {
		var err error
		traffics, clientTraffics, err = j.xrayService.GetXrayTraffic()
		if err == nil {
			collected = true
			invalidCount := j.pending.add(traffics, clientTraffics)
			if invalidCount > 0 {
				logger.Warningf("ignored %d invalid Xray traffic delta entries for persistence", invalidCount)
			}
		}
	}

	if !collected && j.pending.empty() {
		return
	}

	needRestart := false
	inboundTraffics, pendingClientTraffics := j.pending.inboundBatch()
	if collected || len(inboundTraffics) > 0 || len(pendingClientTraffics) > 0 {
		err, stepRestart := j.inboundService.AddTraffic(inboundTraffics, pendingClientTraffics)
		needRestart = needRestart || stepRestart
		if err != nil {
			logger.Warningf(
				"add inbound traffic failed; retaining %d inbound and %d client deltas: %v",
				len(inboundTraffics), len(pendingClientTraffics), err,
			)
		} else {
			j.pending.clearInbound()
		}
	}

	outboundTraffics := j.pending.outboundBatch()
	if collected || len(outboundTraffics) > 0 {
		err, stepRestart := j.outboundService.AddTraffic(outboundTraffics, nil)
		needRestart = needRestart || stepRestart
		if err != nil {
			logger.Warningf("add outbound traffic failed; retaining %d outbound deltas: %v", len(outboundTraffics), err)
		} else {
			j.pending.clearOutbound()
		}
	}
	if needRestart && j.xrayService.IsXrayRunning() {
		j.xrayService.SetToNeedRestart()
	}

	// Retried deltas have already been broadcast when first collected.
	if !collected {
		return
	}

	if ExternalTrafficInformEnable, err := j.settingService.GetExternalTrafficInformEnable(); ExternalTrafficInformEnable {
		j.informTrafficToExternalAPI(traffics, clientTraffics)
	} else if err != nil {
		logger.Warning("get ExternalTrafficInformEnable failed:", err)
	}

	// If no frontend client is connected, skip all WebSocket broadcasting routines,
	// including expensive DB queries for online clients and JSON marshaling.
	if !websocket.HasClients() {
		return
	}

	// Update online clients list and map
	onlineClients := j.inboundService.GetOnlineClients()
	lastOnlineMap, err := j.inboundService.GetClientsLastOnline()
	if err != nil {
		logger.Warning("get clients last online failed:", err)
		lastOnlineMap = make(map[string]int64)
	}

	// Broadcast traffic update (deltas and online stats) via WebSocket
	trafficUpdate := map[string]any{
		"traffics":       traffics,
		"clientTraffics": clientTraffics,
		"onlineClients":  onlineClients,
		"lastOnlineMap":  lastOnlineMap,
	}
	websocket.BroadcastTraffic(trafficUpdate)

	// Fetch updated inbounds from database with accumulated traffic values
	// This ensures frontend receives the actual total traffic for real-time UI refresh.
	updatedInbounds, err := j.inboundService.GetAllInbounds()
	if err != nil {
		logger.Warning("get all inbounds for websocket failed:", err)
	}

	updatedOutbounds, err := j.outboundService.GetOutboundsTraffic()
	if err != nil {
		logger.Warning("get all outbounds for websocket failed:", err)
	}

	// The WebSocket hub will automatically check the payload size.
	// If it exceeds 100MB, it sends a lightweight 'invalidate' signal instead.
	if updatedInbounds != nil {
		websocket.BroadcastInbounds(updatedInbounds)
	}

	if updatedOutbounds != nil {
		websocket.BroadcastOutbounds(updatedOutbounds)
	}
}

func (p *pendingTraffic) add(traffics []*xray.Traffic, clientTraffics []*xray.ClientTraffic) int {
	invalidCount := 0
	for _, traffic := range traffics {
		if traffic == nil || traffic.Tag == "" || traffic.Up < 0 || traffic.Down < 0 || traffic.IsInbound == traffic.IsOutbound {
			invalidCount++
			continue
		}
		if traffic.Up == 0 && traffic.Down == 0 {
			continue
		}
		if traffic.IsInbound {
			if p.inbounds == nil {
				p.inbounds = make(map[string]trafficDelta)
			}
			p.inbounds[traffic.Tag] = mergeTrafficDelta(p.inbounds[traffic.Tag], traffic.Up, traffic.Down)
		} else {
			if p.outbounds == nil {
				p.outbounds = make(map[string]trafficDelta)
			}
			p.outbounds[traffic.Tag] = mergeTrafficDelta(p.outbounds[traffic.Tag], traffic.Up, traffic.Down)
		}
	}

	for _, traffic := range clientTraffics {
		if traffic == nil || traffic.Email == "" || traffic.Up < 0 || traffic.Down < 0 {
			invalidCount++
			continue
		}
		if traffic.Up == 0 && traffic.Down == 0 {
			continue
		}
		if p.clients == nil {
			p.clients = make(map[string]trafficDelta)
		}
		p.clients[traffic.Email] = mergeTrafficDelta(p.clients[traffic.Email], traffic.Up, traffic.Down)
	}
	return invalidCount
}

func mergeTrafficDelta(current trafficDelta, up, down int64) trafficDelta {
	current.up += up
	current.down += down
	return current
}

func (p *pendingTraffic) empty() bool {
	return len(p.inbounds) == 0 && len(p.clients) == 0 && len(p.outbounds) == 0
}

func (p *pendingTraffic) inboundBatch() ([]*xray.Traffic, []*xray.ClientTraffic) {
	inbounds := make([]*xray.Traffic, 0, len(p.inbounds))
	for tag, delta := range p.inbounds {
		inbounds = append(inbounds, &xray.Traffic{
			IsInbound: true,
			Tag:       tag,
			Up:        delta.up,
			Down:      delta.down,
		})
	}
	clients := make([]*xray.ClientTraffic, 0, len(p.clients))
	for email, delta := range p.clients {
		clients = append(clients, &xray.ClientTraffic{
			Email: email,
			Up:    delta.up,
			Down:  delta.down,
		})
	}
	return inbounds, clients
}

func (p *pendingTraffic) outboundBatch() []*xray.Traffic {
	outbounds := make([]*xray.Traffic, 0, len(p.outbounds))
	for tag, delta := range p.outbounds {
		outbounds = append(outbounds, &xray.Traffic{
			IsOutbound: true,
			Tag:        tag,
			Up:         delta.up,
			Down:       delta.down,
		})
	}
	return outbounds
}

func (p *pendingTraffic) clearInbound() {
	clear(p.inbounds)
	clear(p.clients)
}

func (p *pendingTraffic) clearOutbound() {
	clear(p.outbounds)
}

func (j *XrayTrafficJob) informTrafficToExternalAPI(inboundTraffics []*xray.Traffic, clientTraffics []*xray.ClientTraffic) {
	informURL, err := j.settingService.GetExternalTrafficInformURI()
	if err != nil {
		logger.Warning("get ExternalTrafficInformURI failed:", err)
		return
	}
	requestBody, err := json.Marshal(map[string]any{"clientTraffics": clientTraffics, "inboundTraffics": inboundTraffics})
	if err != nil {
		logger.Warning("parse client/inbound traffic failed:", err)
		return
	}
	request := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(request)
	request.Header.SetMethod("POST")
	request.Header.SetContentType("application/json; charset=UTF-8")
	request.SetBody([]byte(requestBody))
	request.SetRequestURI(informURL)
	response := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(response)
	if err := fasthttp.Do(request, response); err != nil {
		logger.Warning("POST ExternalTrafficInformURI failed:", err)
	}
}
