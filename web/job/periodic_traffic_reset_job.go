package job

import (
	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/logger"
	"github.com/superaddmin/SuperXray-gui/v2/web/service"
)

// Period represents the time period for traffic resets.
type Period string

type periodicTrafficResetInboundService interface {
	GetInboundsByTrafficReset(string) ([]*model.Inbound, error)
	ResetInboundTraffic(int) error
	ResetAllClientTraffics(int) error
}

type periodicTrafficResetXrayService interface {
	IsXrayRunning() bool
	SetToNeedRestart()
}

// PeriodicTrafficResetJob resets traffic statistics for inbounds based on their configured reset period.
type PeriodicTrafficResetJob struct {
	inboundService periodicTrafficResetInboundService
	xrayService    periodicTrafficResetXrayService
	period         Period
}

// NewPeriodicTrafficResetJob creates a new periodic traffic reset job for the specified period.
func NewPeriodicTrafficResetJob(period Period) *PeriodicTrafficResetJob {
	return &PeriodicTrafficResetJob{
		inboundService: &service.InboundService{},
		xrayService:    &service.XrayService{},
		period:         period,
	}
}

// Run resets traffic statistics for all inbounds that match the configured reset period.
func (j *PeriodicTrafficResetJob) Run() {
	inbounds, err := j.inboundService.GetInboundsByTrafficReset(string(j.period))
	if err != nil {
		logger.Warning("Failed to get inbounds for traffic reset:", err)
		return
	}

	if len(inbounds) == 0 {
		return
	}
	logger.Infof("Running periodic traffic reset job for period: %s (%d matching inbounds)", j.period, len(inbounds))

	resetCount := 0
	trafficChanged := false

	for _, inbound := range inbounds {
		resetInboundErr := j.inboundService.ResetInboundTraffic(inbound.Id)
		if resetInboundErr != nil {
			logger.Warning("Failed to reset traffic for inbound", inbound.Id, ":", resetInboundErr)
		}

		resetClientErr := j.inboundService.ResetAllClientTraffics(inbound.Id)
		if resetClientErr != nil {
			logger.Warning("Failed to reset traffic for all users of inbound", inbound.Id, ":", resetClientErr)
		}

		if resetInboundErr == nil && resetClientErr == nil {
			resetCount++
		}
		// Each operation commits independently; either success can restore
		// persisted state that needs to be synchronized with the running core.
		trafficChanged = trafficChanged || resetInboundErr == nil || resetClientErr == nil
	}

	if trafficChanged {
		// Resetting persisted traffic also has to restore clients/inbounds
		// previously removed from the running core. Respect a manual stop.
		if j.xrayService.IsXrayRunning() {
			j.xrayService.SetToNeedRestart()
		}
	}
	if resetCount > 0 {
		logger.Infof("Periodic traffic reset completed: %d inbounds reset", resetCount)
	}
}
