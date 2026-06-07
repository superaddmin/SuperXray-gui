package sub

import (
	"fmt"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
)

// SubscriptionDiagnostic 表示订阅诊断接口返回的节点生成统计和可读提示。
type SubscriptionDiagnostic struct {
	SubID              string                       `json:"subId"`
	Target             string                       `json:"target"`
	Format             string                       `json:"format"`
	SupportedFormats   []string                     `json:"supportedFormats"`
	SupportedProtocols []string                     `json:"supportedProtocols"`
	TotalInbounds      int                          `json:"totalInbounds"`
	OutputNodes        int                          `json:"outputNodes"`
	SkippedNodes       int                          `json:"skippedNodes"`
	Warnings           []string                     `json:"warnings"`
	SkipReasons        []SubscriptionDiagnosticSkip `json:"skipReasons"`
}

// SubscriptionDiagnosticSkip 表示单个入站在订阅诊断中被跳过的协议和原因。
type SubscriptionDiagnosticSkip struct {
	InboundID int    `json:"inboundId"`
	Protocol  string `json:"protocol"`
	Reason    string `json:"reason"`
}

// diagnoseSubscriptionInbounds 根据订阅格式统计入站可输出节点数、跳过数量和可读原因。
func diagnoseSubscriptionInbounds(inbounds []*model.Inbound, subId string, format subscriptionFormat) SubscriptionDiagnostic {
	diagnostic := SubscriptionDiagnostic{
		SubID:              subId,
		Format:             string(format),
		SupportedFormats:   supportedSubscriptionFormats(),
		SupportedProtocols: supportedSubscriptionProtocols(),
		TotalInbounds:      len(inbounds),
	}
	if len(inbounds) == 0 {
		diagnostic.Warnings = append(diagnostic.Warnings, fmt.Sprintf("No enabled subscription inbounds found for subId %q", subId))
		return diagnostic
	}

	service := &SubService{remarkModel: "-ieo"}
	for _, inbound := range inbounds {
		if inbound == nil {
			diagnostic.SkippedNodes++
			diagnostic.SkipReasons = append(diagnostic.SkipReasons, SubscriptionDiagnosticSkip{Reason: "inbound is nil"})
			continue
		}
		supported := slicesContainsProtocol(subscriptionClientProtocols(), inbound.Protocol) || slicesContainsProtocol(subscriptionPeerProtocols(), inbound.Protocol)
		if !supported {
			diagnostic.SkippedNodes++
			diagnostic.SkipReasons = append(diagnostic.SkipReasons, SubscriptionDiagnosticSkip{
				InboundID: inbound.Id,
				Protocol:  string(inbound.Protocol),
				Reason:    "protocol is not supported by subscription output",
			})
			continue
		}

		if inbound.Protocol == model.WireGuard {
			peers, err := wireguardPeersBySubID(inbound, subId)
			if err != nil || len(peers) == 0 {
				diagnostic.SkippedNodes++
				diagnostic.SkipReasons = append(diagnostic.SkipReasons, SubscriptionDiagnosticSkip{
					InboundID: inbound.Id,
					Protocol:  string(inbound.Protocol),
					Reason:    "no enabled WireGuard peer matches this subscription id",
				})
				continue
			}
			diagnostic.OutputNodes += len(peers)
			continue
		}

		clients, err := service.inboundService.GetClients(inbound)
		if err != nil || len(clients) == 0 {
			diagnostic.SkippedNodes++
			diagnostic.SkipReasons = append(diagnostic.SkipReasons, SubscriptionDiagnosticSkip{
				InboundID: inbound.Id,
				Protocol:  string(inbound.Protocol),
				Reason:    "no readable clients found in inbound settings",
			})
			continue
		}
		matched := 0
		for _, client := range clients {
			if client.Enable && client.SubID == subId {
				matched++
			}
		}
		if matched == 0 {
			diagnostic.SkippedNodes++
			diagnostic.SkipReasons = append(diagnostic.SkipReasons, SubscriptionDiagnosticSkip{
				InboundID: inbound.Id,
				Protocol:  string(inbound.Protocol),
				Reason:    "no enabled client matches this subscription id",
			})
			continue
		}
		diagnostic.OutputNodes += matched
	}

	if diagnostic.OutputNodes == 0 {
		diagnostic.Warnings = append(diagnostic.Warnings, "Subscription generation would produce no output nodes")
	}
	if diagnostic.SkippedNodes > 0 {
		diagnostic.Warnings = append(diagnostic.Warnings, fmt.Sprintf("%d inbound(s) were skipped during subscription generation", diagnostic.SkippedNodes))
	}
	return diagnostic
}

func supportedSubscriptionFormats() []string {
	return []string{
		string(subscriptionFormatURI),
		string(subscriptionFormatJSON),
		string(subscriptionFormatClash),
		string(subscriptionFormatWireGuard),
	}
}

func supportedSubscriptionProtocols() []string {
	protocols := make([]string, 0, len(subscriptionClientProtocols())+len(subscriptionPeerProtocols()))
	seen := make(map[string]bool)
	for _, protocol := range append(subscriptionClientProtocols(), subscriptionPeerProtocols()...) {
		value := string(protocol)
		if seen[value] {
			continue
		}
		seen[value] = true
		protocols = append(protocols, value)
	}
	return protocols
}

func slicesContainsProtocol(protocols []model.Protocol, protocol model.Protocol) bool {
	for _, item := range protocols {
		if item == protocol {
			return true
		}
	}
	return false
}
