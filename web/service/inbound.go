// Package service provides business logic services for the SuperXray web panel,
// including inbound/outbound management, user administration, settings, and Xray integration.
package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/superaddmin/SuperXray-gui/v2/database"
	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/logger"
	"github.com/superaddmin/SuperXray-gui/v2/util/common"
	"github.com/superaddmin/SuperXray-gui/v2/xray"

	"gorm.io/gorm"
)

// InboundService provides business logic for managing Xray inbound configurations.
// It handles CRUD operations for inbounds, client management, traffic monitoring,
// and integration with the Xray API for real-time updates.
type InboundService struct {
	xrayApi           xray.XrayAPI
	inboundRepository database.InboundRepository
}

func NewInboundService(inboundRepository database.InboundRepository) *InboundService {
	return &InboundService{
		inboundRepository: inboundRepository,
	}
}

func (s *InboundService) inbounds() database.InboundRepository {
	if s.inboundRepository != nil {
		return s.inboundRepository
	}
	return database.NewRepositories(database.GetDB()).Inbounds
}

var xrayAPISyncMu sync.Mutex

func withXrayAPISyncLock(fn func()) {
	xrayAPISyncMu.Lock()
	defer xrayAPISyncMu.Unlock()
	fn()
}

func xrayRuntimeProtocol(protocol model.Protocol) string {
	return string(protocol.XrayProtocol())
}

type CopyClientsResult struct {
	Added   []string `json:"added"`
	Skipped []string `json:"skipped"`
	Errors  []string `json:"errors"`
}

// GetInbounds retrieves all inbounds for a specific user.
// Returns a slice of inbound models with their associated client statistics.
func (s *InboundService) GetInbounds(userId int) ([]*model.Inbound, error) {
	inbounds, err := s.inbounds().ListByUserID(userId)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	s.enrichClientStats(inbounds)
	return inbounds, nil
}

type InboundOption struct {
	Id             int    `json:"id" example:"1"`
	Remark         string `json:"remark" example:"VLESS-443"`
	Tag            string `json:"tag" example:"in-443-tcp"`
	Protocol       string `json:"protocol" example:"vless"`
	Port           int    `json:"port" example:"443"`
	TlsFlowCapable bool   `json:"tlsFlowCapable" example:"true"`
	SsMethod       string `json:"ssMethod"`
}

func (s *InboundService) GetInboundOptions(userId int) ([]InboundOption, error) {
	records, err := s.inbounds().ListOptionsByUserID(userId)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	out := make([]InboundOption, 0, len(records))
	for _, record := range records {
		out = append(out, InboundOption{
			Id:             record.ID,
			Remark:         record.Remark,
			Tag:            record.Tag,
			Protocol:       record.Protocol,
			Port:           record.Port,
			TlsFlowCapable: record.TLSFlowCapable,
			SsMethod:       record.SSMethod,
		})
	}
	return out, nil
}

// GetAllInbounds retrieves all inbounds from the database.
// Returns a slice of all inbound models with their associated client statistics.
func (s *InboundService) GetAllInbounds() ([]*model.Inbound, error) {
	inbounds, err := s.inbounds().ListAll()
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	s.enrichClientStats(inbounds)
	return inbounds, nil
}

func (s *InboundService) enrichClientStats(inbounds []*model.Inbound) {
	for _, inbound := range inbounds {
		clients, _ := s.GetClients(inbound)
		if len(clients) == 0 || len(inbound.ClientStats) == 0 {
			continue
		}
		cMap := make(map[string]model.Client, len(clients))
		for _, c := range clients {
			cMap[strings.ToLower(c.Email)] = c
		}
		for i := range inbound.ClientStats {
			email := strings.ToLower(inbound.ClientStats[i].Email)
			if c, ok := cMap[email]; ok {
				inbound.ClientStats[i].UUID = c.ID
				inbound.ClientStats[i].SubId = c.SubID
			}
		}
	}
}

func (s *InboundService) GetInboundsByTrafficReset(period string) ([]*model.Inbound, error) {
	inbounds, err := s.inbounds().ListByTrafficReset(period)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return inbounds, nil
}

func (s *InboundService) checkPortExist(listen string, port int, ignoreId int) (bool, error) {
	return s.inbounds().PortExists(listen, port, ignoreId)
}

func (s *InboundService) GetClients(inbound *model.Inbound) ([]model.Client, error) {
	clients, err := parseInboundClients(inbound.Settings)
	if err != nil {
		return nil, err
	}
	return clients, nil
}

func parseInboundClients(rawSettings string) ([]model.Client, error) {
	var settings *struct {
		Clients []model.Client `json:"clients"`
	}
	if err := json.Unmarshal([]byte(rawSettings), &settings); err != nil {
		return nil, err
	}
	if settings == nil {
		return nil, fmt.Errorf("setting is null")
	}

	if settings.Clients == nil {
		return nil, nil
	}
	return settings.Clients, nil
}

func parseInboundSettingsEntry(rawSettings string) (map[string]any, []any, error) {
	var settings map[string]any
	if err := json.Unmarshal([]byte(rawSettings), &settings); err != nil {
		return nil, nil, err
	}
	if settings == nil {
		return nil, nil, fmt.Errorf("settings is null")
	}
	rawClients, ok := settings["clients"]
	if !ok {
		return nil, nil, fmt.Errorf("settings.clients is required")
	}
	clients, ok := rawClients.([]any)
	if !ok {
		return nil, nil, fmt.Errorf("settings.clients must be an array")
	}
	return settings, clients, nil
}

func (s *InboundService) getAllEmails() ([]string, error) {
	return s.inbounds().ListClientEmails()
}

func (s *InboundService) contains(slice []string, str string) bool {
	lowerStr := strings.ToLower(str)
	for _, s := range slice {
		if strings.ToLower(s) == lowerStr {
			return true
		}
	}
	return false
}

func (s *InboundService) checkEmailsExistForClients(clients []model.Client) (string, error) {
	allEmails, err := s.getAllEmails()
	if err != nil {
		return "", err
	}
	var emails []string
	for _, client := range clients {
		if client.Email != "" {
			if s.contains(emails, client.Email) {
				return client.Email, nil
			}
			if s.contains(allEmails, client.Email) {
				return client.Email, nil
			}
			emails = append(emails, client.Email)
		}
	}
	return "", nil
}

func (s *InboundService) checkEmailExistForInbound(inbound *model.Inbound) (string, error) {
	clients, err := s.GetClients(inbound)
	if err != nil {
		return "", err
	}
	allEmails, err := s.getAllEmails()
	if err != nil {
		return "", err
	}
	var emails []string
	for _, client := range clients {
		if client.Email != "" {
			if s.contains(emails, client.Email) {
				return client.Email, nil
			}
			if s.contains(allEmails, client.Email) {
				return client.Email, nil
			}
			emails = append(emails, client.Email)
		}
	}
	return "", nil
}

// AddInbound creates a new inbound configuration.
// It validates port uniqueness, client email uniqueness, and required fields,
// then saves the inbound to the database and optionally adds it to the running Xray instance.
// Returns the created inbound, whether Xray needs restart, and any error.
func (s *InboundService) AddInbound(inbound *model.Inbound) (*model.Inbound, bool, error) {
	exist, err := s.checkPortExist(inbound.Listen, inbound.Port, 0)
	if err != nil {
		return inbound, false, err
	}
	if exist {
		return inbound, false, common.NewError("Port already exists:", inbound.Port)
	}
	if err := normalizeShadowsocksInboundSettings(inbound); err != nil {
		return inbound, false, err
	}

	existEmail, err := s.checkEmailExistForInbound(inbound)
	if err != nil {
		return inbound, false, err
	}
	if existEmail != "" {
		return inbound, false, common.NewError("Duplicate email:", existEmail)
	}

	clients, err := s.GetClients(inbound)
	if err != nil {
		return inbound, false, err
	}
	if err := validateInboundProtocolConfig(inbound); err != nil {
		return inbound, false, err
	}

	// Ensure created_at and updated_at on clients in settings
	if len(clients) > 0 {
		var settings map[string]any
		if err2 := json.Unmarshal([]byte(inbound.Settings), &settings); err2 == nil && settings != nil {
			now := time.Now().Unix() * 1000
			updatedClients := make([]model.Client, 0, len(clients))
			for _, c := range clients {
				if c.CreatedAt == 0 {
					c.CreatedAt = now
				}
				c.UpdatedAt = now
				updatedClients = append(updatedClients, c)
			}
			settings["clients"] = updatedClients
			if bs, err3 := json.MarshalIndent(settings, "", "  "); err3 == nil {
				inbound.Settings = string(bs)
			} else {
				logger.Debug("Unable to marshal inbound settings with timestamps:", err3)
			}
		} else if err2 != nil {
			logger.Debug("Unable to parse inbound settings for timestamps:", err2)
		}
	}

	// Secure client ID
	for _, client := range clients {
		switch inbound.Protocol {
		case "trojan":
			if client.Password == "" {
				return inbound, false, common.NewError("empty client ID")
			}
		case "shadowsocks":
			if client.Email == "" {
				return inbound, false, common.NewError("empty client ID")
			}
		case "hysteria", "hysteria2":
			if client.Auth == "" {
				return inbound, false, common.NewError("empty client ID")
			}
		default:
			if client.ID == "" {
				return inbound, false, common.NewError("empty client ID")
			}
		}
	}

	db := database.GetDB()
	tx := db.Begin()
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	err = s.inbounds().SaveInbound(tx, inbound)
	if err == nil {
		if len(inbound.ClientStats) == 0 {
			for _, client := range clients {
				if err := s.AddClientStat(tx, inbound.Id, &client); err != nil {
					return inbound, false, err
				}
			}
		}
	} else {
		return inbound, false, err
	}

	needRestart := false
	if inbound.Enable {
		withXrayAPISyncLock(func() {
			if err1 := s.xrayApi.Init(p.GetAPIPort()); err1 != nil {
				logger.Debug("Unable to init xray api:", err1)
				needRestart = true
			}
			inboundJson, err1 := json.MarshalIndent(inbound.GenXrayInboundConfig(), "", "  ")
			if err1 != nil {
				logger.Debug("Unable to marshal inbound config:", err1)
			}

			if !needRestart {
				err1 = s.xrayApi.AddInbound(inboundJson)
				if err1 == nil {
					logger.Debug("New inbound added by api:", inbound.Tag)
				} else {
					logger.Debug("Unable to add inbound by api:", err1)
					needRestart = true
				}
			}
			s.xrayApi.Close()
		})
	}

	return inbound, needRestart, err
}

// DelInbound deletes an inbound configuration by ID.
// It removes the inbound from the database and the running Xray instance if active.
// Returns whether Xray needs restart and any error.
func (s *InboundService) DelInbound(id int) (bool, error) {
	db := database.GetDB()

	inbound, err := s.GetInbound(id)
	if err != nil {
		return false, err
	}

	tag := inbound.Tag
	needRestart := false
	if inbound.Enable {
		withXrayAPISyncLock(func() {
			if err1 := s.xrayApi.Init(p.GetAPIPort()); err1 != nil {
				logger.Debug("Unable to init xray api:", err1)
				needRestart = true
			} else {
				err1 := s.xrayApi.DelInbound(tag)
				if err1 == nil {
					logger.Debug("Inbound deleted by api:", tag)
				} else {
					logger.Debug("Unable to delete inbound by api:", err1)
					needRestart = true
				}
				s.xrayApi.Close()
			}
		})
	} else {
		logger.Debug("No enabled inbound founded to removing by api", tag)
	}

	// Delete client traffics of inbounds
	err = s.inbounds().DeleteClientTrafficsByInboundID(db, id)
	if err != nil {
		return false, err
	}
	clients, err := s.GetClients(inbound)
	if err != nil {
		return false, err
	}
	for _, client := range clients {
		err := s.DelClientIPs(db, client.Email)
		if err != nil {
			return false, err
		}
	}

	return needRestart, s.inbounds().DeleteByID(db, id)
}

func (s *InboundService) GetInbound(id int) (*model.Inbound, error) {
	return s.inbounds().Get(id)
}

// UpdateInbound modifies an existing inbound configuration.
// It validates changes, updates the database, and syncs with the running Xray instance.
// Returns the updated inbound, whether Xray needs restart, and any error.
func (s *InboundService) UpdateInbound(inbound *model.Inbound) (*model.Inbound, bool, error) {
	exist, err := s.checkPortExist(inbound.Listen, inbound.Port, inbound.Id)
	if err != nil {
		return inbound, false, err
	}
	if exist {
		return inbound, false, common.NewError("Port already exists:", inbound.Port)
	}

	oldInbound, err := s.GetInbound(inbound.Id)
	if err != nil {
		return inbound, false, err
	}
	if err := normalizeShadowsocksInboundSettings(inbound); err != nil {
		return inbound, false, err
	}
	if err := validateInboundProtocolConfig(inbound); err != nil {
		return inbound, false, err
	}

	tag := oldInbound.Tag

	db := database.GetDB()
	tx := db.Begin()

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	err = s.updateClientTraffics(tx, oldInbound, inbound)
	if err != nil {
		return inbound, false, err
	}

	// Ensure created_at and updated_at exist in inbound.Settings clients
	{
		var oldSettings map[string]any
		_ = json.Unmarshal([]byte(oldInbound.Settings), &oldSettings)
		emailToCreated := map[string]int64{}
		emailToUpdated := map[string]int64{}
		if oldSettings != nil {
			if oc, ok := oldSettings["clients"].([]any); ok {
				for _, it := range oc {
					if m, ok2 := it.(map[string]any); ok2 {
						if email, ok3 := m["email"].(string); ok3 {
							switch v := m["created_at"].(type) {
							case float64:
								emailToCreated[email] = int64(v)
							case int64:
								emailToCreated[email] = v
							}
							switch v := m["updated_at"].(type) {
							case float64:
								emailToUpdated[email] = int64(v)
							case int64:
								emailToUpdated[email] = v
							}
						}
					}
				}
			}
		}
		var newSettings map[string]any
		if err2 := json.Unmarshal([]byte(inbound.Settings), &newSettings); err2 == nil && newSettings != nil {
			now := time.Now().Unix() * 1000
			if nSlice, ok := newSettings["clients"].([]any); ok {
				for i := range nSlice {
					if m, ok2 := nSlice[i].(map[string]any); ok2 {
						email, _ := m["email"].(string)
						if _, ok3 := m["created_at"]; !ok3 {
							if v, ok4 := emailToCreated[email]; ok4 && v > 0 {
								m["created_at"] = v
							} else {
								m["created_at"] = now
							}
						}
						// Preserve client's updated_at if present; do not bump on parent inbound update
						if _, hasUpdated := m["updated_at"]; !hasUpdated {
							if v, ok4 := emailToUpdated[email]; ok4 && v > 0 {
								m["updated_at"] = v
							}
						}
						nSlice[i] = m
					}
				}
				newSettings["clients"] = nSlice
				if bs, err3 := json.MarshalIndent(newSettings, "", "  "); err3 == nil {
					inbound.Settings = string(bs)
				}
			}
		}
	}

	oldInbound.Up = inbound.Up
	oldInbound.Down = inbound.Down
	oldInbound.Total = inbound.Total
	oldInbound.Remark = inbound.Remark
	oldInbound.Enable = inbound.Enable
	oldInbound.ExpiryTime = inbound.ExpiryTime
	oldInbound.TrafficReset = inbound.TrafficReset
	oldInbound.Listen = inbound.Listen
	oldInbound.Port = inbound.Port
	oldInbound.Protocol = inbound.Protocol
	oldInbound.Settings = inbound.Settings
	oldInbound.StreamSettings = inbound.StreamSettings
	oldInbound.Sniffing = inbound.Sniffing
	if inbound.Listen == "" || inbound.Listen == "0.0.0.0" || inbound.Listen == "::" || inbound.Listen == "::0" {
		oldInbound.Tag = fmt.Sprintf("inbound-%v", inbound.Port)
	} else {
		oldInbound.Tag = fmt.Sprintf("inbound-%v:%v", inbound.Listen, inbound.Port)
	}

	needRestart := false
	withXrayAPISyncLock(func() {
		if err2 := s.xrayApi.Init(p.GetAPIPort()); err2 != nil {
			logger.Debug("Unable to init xray api:", err2)
			needRestart = true
		} else {
			if s.xrayApi.DelInbound(tag) == nil {
				logger.Debug("Old inbound deleted by api:", tag)
			}
			if inbound.Enable {
				runtimeInbound, err2 := s.buildRuntimeInboundForAPI(tx, oldInbound)
				if err2 != nil {
					logger.Debug("Unable to prepare runtime inbound config:", err2)
					needRestart = true
				} else {
					inboundJson, err2 := json.MarshalIndent(runtimeInbound.GenXrayInboundConfig(), "", "  ")
					if err2 != nil {
						logger.Debug("Unable to marshal updated inbound config:", err2)
						needRestart = true
					} else {
						err2 = s.xrayApi.AddInbound(inboundJson)
						if err2 == nil {
							logger.Debug("Updated inbound added by api:", oldInbound.Tag)
						} else {
							logger.Debug("Unable to update inbound by api:", err2)
							needRestart = true
						}
					}
				}
			}
			s.xrayApi.Close()
		}
	})

	return inbound, needRestart, s.inbounds().SaveInbound(tx, oldInbound)
}

func (s *InboundService) buildRuntimeInboundForAPI(tx *gorm.DB, inbound *model.Inbound) (*model.Inbound, error) {
	if inbound == nil {
		return nil, fmt.Errorf("inbound is nil")
	}

	runtimeInbound := *inbound
	settings := map[string]any{}
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		return nil, err
	}

	clients, ok := settings["clients"].([]any)
	if !ok {
		return &runtimeInbound, nil
	}

	clientStats, err := s.inbounds().ListClientTrafficEnableByInboundID(tx, inbound.Id)
	if err != nil {
		return nil, err
	}

	enableMap := make(map[string]bool, len(clientStats))
	for _, clientTraffic := range clientStats {
		enableMap[clientTraffic.Email] = clientTraffic.Enable
	}

	finalClients := make([]any, 0, len(clients))
	for _, client := range clients {
		c, ok := client.(map[string]any)
		if !ok {
			continue
		}

		email, _ := c["email"].(string)
		if enable, exists := enableMap[email]; exists && !enable {
			continue
		}

		if manualEnable, ok := c["enable"].(bool); ok && !manualEnable {
			continue
		}

		finalClients = append(finalClients, c)
	}

	settings["clients"] = finalClients
	modifiedSettings, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return nil, err
	}
	runtimeInbound.Settings = string(modifiedSettings)

	return &runtimeInbound, nil
}

func (s *InboundService) updateClientTraffics(tx *gorm.DB, oldInbound *model.Inbound, newInbound *model.Inbound) error {
	oldClients, err := s.GetClients(oldInbound)
	if err != nil {
		return err
	}
	newClients, err := s.GetClients(newInbound)
	if err != nil {
		return err
	}

	var emailExists bool

	for _, oldClient := range oldClients {
		emailExists = false
		for _, newClient := range newClients {
			if oldClient.Email == newClient.Email {
				emailExists = true
				break
			}
		}
		if !emailExists {
			err = s.DelClientStat(tx, oldClient.Email)
			if err != nil {
				return err
			}
		}
	}
	for _, newClient := range newClients {
		emailExists = false
		for _, oldClient := range oldClients {
			if newClient.Email == oldClient.Email {
				emailExists = true
				break
			}
		}
		if !emailExists {
			err = s.AddClientStat(tx, oldInbound.Id, &newClient)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *InboundService) AddInboundClient(data *model.Inbound) (bool, error) {
	var interfaceClients []any
	var err error
	_, interfaceClients, err = parseInboundSettingsEntry(data.Settings)
	if err != nil {
		return false, err
	}

	oldInbound, err := s.GetInbound(data.Id)
	if err != nil {
		return false, err
	}
	if oldInbound.Protocol == model.Shadowsocks {
		normalizeShadowsocksClientEntries(shadowsocksMethodFromSettings(oldInbound.Settings), interfaceClients)
		normalizedPayload, err := json.Marshal(map[string][]any{"clients": interfaceClients})
		if err != nil {
			return false, err
		}
		data.Settings = string(normalizedPayload)
	}

	clients, err := s.GetClients(data)
	if err != nil {
		return false, err
	}

	// Add timestamps for new clients being appended
	nowTs := time.Now().Unix() * 1000
	for i := range interfaceClients {
		if cm, ok := interfaceClients[i].(map[string]any); ok {
			if _, ok2 := cm["created_at"]; !ok2 {
				cm["created_at"] = nowTs
			}
			cm["updated_at"] = nowTs
			interfaceClients[i] = cm
		}
	}
	existEmail, err := s.checkEmailsExistForClients(clients)
	if err != nil {
		return false, err
	}
	if existEmail != "" {
		return false, common.NewError("Duplicate email:", existEmail)
	}

	if err := validateInboundProtocolClients(oldInbound.Protocol, oldInbound.Settings, oldInbound.StreamSettings, clients); err != nil {
		return false, err
	}

	// Secure client ID
	for _, client := range clients {
		switch oldInbound.Protocol {
		case "trojan":
			if client.Password == "" {
				return false, common.NewError("empty client ID")
			}
		case "shadowsocks":
			if client.Email == "" {
				return false, common.NewError("empty client ID")
			}
		case "hysteria", "hysteria2":
			if client.Auth == "" {
				return false, common.NewError("empty client ID")
			}
		default:
			if client.ID == "" {
				return false, common.NewError("empty client ID")
			}
		}
	}

	var oldSettings map[string]any
	var oldClients []any
	oldSettings, oldClients, err = parseInboundSettingsEntry(oldInbound.Settings)
	if err != nil {
		return false, err
	}

	oldClients = append(oldClients, interfaceClients...)

	oldSettings["clients"] = oldClients
	if oldInbound.Protocol == model.Shadowsocks {
		normalizeShadowsocksSettingsMap(oldSettings)
	}

	newSettings, err := json.MarshalIndent(oldSettings, "", "  ")
	if err != nil {
		return false, err
	}

	oldInbound.Settings = string(newSettings)

	db := database.GetDB()
	tx := db.Begin()

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	needRestart := false
	withXrayAPISyncLock(func() {
		if err1 := s.xrayApi.Init(p.GetAPIPort()); err1 != nil {
			logger.Debug("Unable to init xray api:", err1)
			needRestart = true
		} else {
			defer s.xrayApi.Close()
		}
		for _, client := range clients {
			if len(client.Email) > 0 {
				err = s.AddClientStat(tx, data.Id, &client)
				if err != nil {
					return
				}
				if client.Enable && !needRestart {
					cipher := ""
					if oldInbound.Protocol == "shadowsocks" {
						cipher, _ = oldSettings["method"].(string)
					}
					err1 := s.xrayApi.AddUser(xrayRuntimeProtocol(oldInbound.Protocol), oldInbound.Tag, map[string]any{
						"email":    client.Email,
						"id":       client.ID,
						"auth":     client.Auth,
						"security": client.Security,
						"flow":     client.Flow,
						"password": client.Password,
						"cipher":   cipher,
					})
					if err1 == nil {
						logger.Debug("Client added by api:", client.Email)
					} else {
						logger.Debug("Error in adding client by api:", err1)
						needRestart = true
					}
				}
			} else {
				needRestart = true
			}
		}
	})
	if err != nil {
		return false, err
	}

	return needRestart, s.inbounds().SaveInbound(tx, oldInbound)
}

func (s *InboundService) getClientPrimaryKey(protocol model.Protocol, client model.Client) string {
	switch protocol {
	case model.Trojan:
		return client.Password
	case model.Shadowsocks:
		return client.Email
	case model.Hysteria, model.Hysteria2:
		return client.Auth
	default:
		return client.ID
	}
}

func (s *InboundService) getClientPrimaryKeyByEmail(protocol model.Protocol, clients []model.Client, email string) string {
	for _, client := range clients {
		if client.Email == email {
			return s.getClientPrimaryKey(protocol, client)
		}
	}
	return ""
}

func (s *InboundService) writeBackClientSubID(sourceInboundID int, sourceProtocol model.Protocol, client model.Client, subID string) (bool, error) {
	client.SubID = subID
	client.UpdatedAt = time.Now().UnixMilli()
	clientID := s.getClientPrimaryKey(sourceProtocol, client)
	if clientID == "" {
		return false, common.NewError("empty client ID")
	}

	// #nosec G117 -- Xray client settings must include protocol credential fields in trusted config JSON.
	settingsBytes, err := json.Marshal(map[string][]model.Client{
		"clients": {client},
	})
	if err != nil {
		return false, err
	}

	updatePayload := &model.Inbound{
		Id:       sourceInboundID,
		Settings: string(settingsBytes),
	}
	return s.UpdateInboundClient(updatePayload, clientID)
}

func (s *InboundService) generateRandomCredential(targetProtocol model.Protocol) string {
	switch targetProtocol {
	case model.VMESS, model.VLESS:
		return uuid.NewString()
	default:
		return randomURLSafeCredential(generatedCredentialBytes)
	}
}

func (s *InboundService) buildTargetClientFromSource(source model.Client, targetInbound *model.Inbound, email string, flow string) (model.Client, error) {
	if targetInbound == nil {
		return model.Client{}, common.NewError("target inbound is required")
	}

	nowTs := time.Now().UnixMilli()
	target := source
	target.Email = email
	target.CreatedAt = nowTs
	target.UpdatedAt = nowTs

	target.ID = ""
	target.Method = ""
	target.Password = ""
	target.Auth = ""
	target.Flow = ""

	targetProtocol := targetInbound.Protocol
	switch targetProtocol {
	case model.VMESS:
		target.ID = s.generateRandomCredential(targetProtocol)
	case model.VLESS:
		target.ID = s.generateRandomCredential(targetProtocol)
		if flow == "xtls-rprx-vision" || flow == "xtls-rprx-vision-udp443" {
			target.Flow = flow
		}
	case model.Trojan:
		target.Password = s.generateRandomCredential(targetProtocol)
	case model.Shadowsocks:
		method := shadowsocksMethodFromSettings(targetInbound.Settings)
		target.Password = randomShadowsocksCredential(method)
		if !isShadowsocks2022Method(method) {
			target.Method = method
		}
	case model.Hysteria, model.Hysteria2:
		target.Auth = s.generateRandomCredential(targetProtocol)
	default:
		target.ID = s.generateRandomCredential(targetProtocol)
	}

	return target, nil
}

func (s *InboundService) nextAvailableCopiedEmail(originalEmail string, targetID int, occupied map[string]struct{}) string {
	base := fmt.Sprintf("%s_%d", originalEmail, targetID)
	candidate := base
	suffix := 0
	for {
		if _, exists := occupied[strings.ToLower(candidate)]; !exists {
			occupied[strings.ToLower(candidate)] = struct{}{}
			return candidate
		}
		suffix++
		candidate = fmt.Sprintf("%s_%d", base, suffix)
	}
}

func (s *InboundService) CopyInboundClients(targetInboundID int, sourceInboundID int, clientEmails []string, flow string) (*CopyClientsResult, bool, error) {
	result := &CopyClientsResult{
		Added:   []string{},
		Skipped: []string{},
		Errors:  []string{},
	}
	if targetInboundID == sourceInboundID {
		return result, false, common.NewError("source and target inbounds must be different")
	}

	targetInbound, err := s.GetInbound(targetInboundID)
	if err != nil {
		return result, false, err
	}
	sourceInbound, err := s.GetInbound(sourceInboundID)
	if err != nil {
		return result, false, err
	}

	sourceClients, err := s.GetClients(sourceInbound)
	if err != nil {
		return result, false, err
	}
	if len(sourceClients) == 0 {
		return result, false, nil
	}

	allowedEmails := map[string]struct{}{}
	if len(clientEmails) > 0 {
		for _, email := range clientEmails {
			allowedEmails[strings.ToLower(strings.TrimSpace(email))] = struct{}{}
		}
	}

	occupiedEmails := map[string]struct{}{}
	allEmails, err := s.getAllEmails()
	if err != nil {
		return result, false, err
	}
	for _, email := range allEmails {
		clean := strings.Trim(email, "\"")
		if clean != "" {
			occupiedEmails[strings.ToLower(clean)] = struct{}{}
		}
	}

	newClients := make([]model.Client, 0)
	needRestart := false
	for _, sourceClient := range sourceClients {
		originalEmail := strings.TrimSpace(sourceClient.Email)
		if originalEmail == "" {
			continue
		}
		if len(allowedEmails) > 0 {
			if _, ok := allowedEmails[strings.ToLower(originalEmail)]; !ok {
				continue
			}
		}

		if sourceClient.SubID == "" {
			newSubID := uuid.NewString()
			subNeedRestart, subErr := s.writeBackClientSubID(sourceInbound.Id, sourceInbound.Protocol, sourceClient, newSubID)
			if subErr != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: failed to write source subId: %v", originalEmail, subErr))
				continue
			}
			if subNeedRestart {
				needRestart = true
			}
			sourceClient.SubID = newSubID
		}

		targetEmail := s.nextAvailableCopiedEmail(originalEmail, targetInboundID, occupiedEmails)
		targetClient, buildErr := s.buildTargetClientFromSource(sourceClient, targetInbound, targetEmail, flow)
		if buildErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", originalEmail, buildErr))
			continue
		}
		newClients = append(newClients, targetClient)
		result.Added = append(result.Added, targetEmail)
	}

	if len(newClients) == 0 {
		return result, needRestart, nil
	}

	// #nosec G117 -- Xray client settings must include protocol credential fields in trusted config JSON.
	settingsPayload, err := json.Marshal(map[string][]model.Client{
		"clients": newClients,
	})
	if err != nil {
		return result, needRestart, err
	}

	addNeedRestart, err := s.AddInboundClient(&model.Inbound{
		Id:       targetInboundID,
		Settings: string(settingsPayload),
	})
	if err != nil {
		return result, needRestart, err
	}
	if addNeedRestart {
		needRestart = true
	}

	return result, needRestart, nil
}

func (s *InboundService) DelInboundClient(inboundId int, clientId string) (bool, error) {
	oldInbound, err := s.GetInbound(inboundId)
	if err != nil {
		logger.Error("Load Old Data Error")
		return false, err
	}
	var settings map[string]any
	var interfaceClients []any
	settings, interfaceClients, err = parseInboundSettingsEntry(oldInbound.Settings)
	if err != nil {
		return false, err
	}

	email := ""
	client_key := "id"
	switch oldInbound.Protocol {
	case "trojan":
		client_key = "password"
	case "shadowsocks":
		client_key = "email"
	case "hysteria", "hysteria2":
		client_key = "auth"
	}

	var newClients []any
	needApiDel := false
	for _, client := range interfaceClients {
		c, ok := client.(map[string]any)
		if !ok {
			newClients = append(newClients, client)
			continue
		}
		cID, _ := c[client_key].(string)
		if cID == clientId {
			email, _ = c["email"].(string)
			needApiDel, _ = c["enable"].(bool)
		} else {
			newClients = append(newClients, client)
		}
	}

	if len(newClients) == 0 {
		return false, common.NewError("no client remained in Inbound")
	}

	settings["clients"] = newClients
	newSettings, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return false, err
	}

	oldInbound.Settings = string(newSettings)

	db := database.GetDB()

	err = s.DelClientIPs(db, email)
	if err != nil {
		logger.Error("Error in delete client IPs")
		return false, err
	}
	needRestart := false

	if len(email) > 0 {
		notDepleted, err := s.inbounds().IsClientTrafficEnabledByEmail(db, email)
		if err != nil {
			logger.Error("Get stats error")
			return false, err
		}
		err = s.DelClientStat(db, email)
		if err != nil {
			logger.Error("Delete stats Data Error")
			return false, err
		}
		if needApiDel && notDepleted {
			withXrayAPISyncLock(func() {
				if err1 := s.xrayApi.Init(p.GetAPIPort()); err1 != nil {
					logger.Debug("Unable to init xray api:", err1)
					needRestart = true
				} else {
					err1 := s.xrayApi.RemoveUser(oldInbound.Tag, email)
					if err1 == nil {
						logger.Debug("Client deleted by api:", email)
						needRestart = false
					} else {
						if strings.Contains(err1.Error(), fmt.Sprintf("User %s not found.", email)) {
							logger.Debug("User is already deleted. Nothing to do more...")
						} else {
							logger.Debug("Error in deleting client by api:", err1)
							needRestart = true
						}
					}
					s.xrayApi.Close()
				}
			})
		}
	}
	return needRestart, s.inbounds().SaveInbound(db, oldInbound)
}

func (s *InboundService) UpdateInboundClient(data *model.Inbound, clientId string) (bool, error) {
	// TODO: check if TrafficReset field is updating
	var interfaceClients []any
	var err error
	_, interfaceClients, err = parseInboundSettingsEntry(data.Settings)
	if err != nil {
		return false, err
	}

	oldInbound, err := s.GetInbound(data.Id)
	if err != nil {
		return false, err
	}
	if oldInbound.Protocol == model.Shadowsocks {
		normalizeShadowsocksClientEntries(shadowsocksMethodFromSettings(oldInbound.Settings), interfaceClients)
		normalizedPayload, err := json.Marshal(map[string][]any{"clients": interfaceClients})
		if err != nil {
			return false, err
		}
		data.Settings = string(normalizedPayload)
	}

	clients, err := s.GetClients(data)
	if err != nil {
		return false, err
	}
	if len(clients) == 0 {
		return false, common.NewError("empty client ID")
	}
	if err := validateInboundProtocolClients(oldInbound.Protocol, oldInbound.Settings, oldInbound.StreamSettings, clients); err != nil {
		return false, err
	}

	oldClients, err := s.GetClients(oldInbound)
	if err != nil {
		return false, err
	}

	oldEmail := ""
	newClientId := ""
	clientIndex := -1
	for index, oldClient := range oldClients {
		oldClientId := ""
		switch oldInbound.Protocol {
		case "trojan":
			oldClientId = oldClient.Password
			newClientId = clients[0].Password
		case "shadowsocks":
			oldClientId = oldClient.Email
			newClientId = clients[0].Email
		case "hysteria", "hysteria2":
			oldClientId = oldClient.Auth
			newClientId = clients[0].Auth
		default:
			oldClientId = oldClient.ID
			newClientId = clients[0].ID
		}
		if clientId == oldClientId {
			oldEmail = oldClient.Email
			clientIndex = index
			break
		}
	}

	// Validate new client ID
	if newClientId == "" || clientIndex == -1 {
		return false, common.NewError("empty client ID")
	}

	if len(clients[0].Email) > 0 && clients[0].Email != oldEmail {
		existEmail, err := s.checkEmailsExistForClients(clients)
		if err != nil {
			return false, err
		}
		if existEmail != "" {
			return false, common.NewError("Duplicate email:", existEmail)
		}
	}

	var oldSettings map[string]any
	var settingsClients []any
	oldSettings, settingsClients, err = parseInboundSettingsEntry(oldInbound.Settings)
	if err != nil {
		return false, err
	}
	// Preserve created_at and set updated_at for the replacing client
	var preservedCreated any
	if clientIndex >= 0 && clientIndex < len(settingsClients) {
		if oldMap, ok := settingsClients[clientIndex].(map[string]any); ok {
			if v, ok2 := oldMap["created_at"]; ok2 {
				preservedCreated = v
			}
		}
	}
	if len(interfaceClients) > 0 {
		if newMap, ok := interfaceClients[0].(map[string]any); ok {
			if preservedCreated == nil {
				preservedCreated = time.Now().Unix() * 1000
			}
			newMap["created_at"] = preservedCreated
			newMap["updated_at"] = time.Now().Unix() * 1000
			interfaceClients[0] = newMap
		}
	}
	settingsClients[clientIndex] = interfaceClients[0]
	oldSettings["clients"] = settingsClients
	if oldInbound.Protocol == model.Shadowsocks {
		normalizeShadowsocksSettingsMap(oldSettings)
	}

	newSettings, err := json.MarshalIndent(oldSettings, "", "  ")
	if err != nil {
		return false, err
	}

	oldInbound.Settings = string(newSettings)
	db := database.GetDB()
	tx := db.Begin()

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	if len(clients[0].Email) > 0 {
		if len(oldEmail) > 0 {
			err = s.UpdateClientStat(tx, oldEmail, &clients[0])
			if err != nil {
				return false, err
			}
			err = s.UpdateClientIPs(tx, oldEmail, clients[0].Email)
			if err != nil {
				return false, err
			}
		} else {
			if err := s.AddClientStat(tx, data.Id, &clients[0]); err != nil {
				return false, err
			}
		}
	} else {
		err = s.DelClientStat(tx, oldEmail)
		if err != nil {
			return false, err
		}
		err = s.DelClientIPs(tx, oldEmail)
		if err != nil {
			return false, err
		}
	}
	needRestart := false
	if len(oldEmail) > 0 {
		withXrayAPISyncLock(func() {
			if err1 := s.xrayApi.Init(p.GetAPIPort()); err1 != nil {
				logger.Debug("Unable to init xray api:", err1)
				needRestart = true
			} else {
				defer s.xrayApi.Close()
			}
			if oldClients[clientIndex].Enable && !needRestart {
				err1 := s.xrayApi.RemoveUser(oldInbound.Tag, oldEmail)
				if err1 == nil {
					logger.Debug("Old client deleted by api:", oldEmail)
				} else {
					if strings.Contains(err1.Error(), fmt.Sprintf("User %s not found.", oldEmail)) {
						logger.Debug("User is already deleted. Nothing to do more...")
					} else {
						logger.Debug("Error in deleting client by api:", err1)
						needRestart = true
					}
				}
			}
			if clients[0].Enable {
				cipher := ""
				if oldInbound.Protocol == "shadowsocks" {
					cipher, _ = oldSettings["method"].(string)
				}
				err1 := s.xrayApi.AddUser(xrayRuntimeProtocol(oldInbound.Protocol), oldInbound.Tag, map[string]any{
					"email":    clients[0].Email,
					"id":       clients[0].ID,
					"security": clients[0].Security,
					"flow":     clients[0].Flow,
					"auth":     clients[0].Auth,
					"password": clients[0].Password,
					"cipher":   cipher,
				})
				if err1 == nil {
					logger.Debug("Client edited by api:", clients[0].Email)
				} else {
					logger.Debug("Error in adding client by api:", err1)
					needRestart = true
				}
			}
		})
	} else {
		logger.Debug("Client old email not found")
		needRestart = true
	}
	return needRestart, s.inbounds().SaveInbound(tx, oldInbound)
}

func (s *InboundService) AddTraffic(inboundTraffics []*xray.Traffic, clientTraffics []*xray.ClientTraffic) (error, bool) {
	var err error
	db := database.GetDB()
	tx := db.Begin()

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()
	err = s.addInboundTraffic(tx, inboundTraffics)
	if err != nil {
		return err, false
	}
	err = s.addClientTraffic(tx, clientTraffics)
	if err != nil {
		return err, false
	}

	needRestart0, count, err := s.autoRenewClients(tx)
	if err != nil {
		logger.Warning("Error in renew clients:", err)
	} else if count > 0 {
		logger.Debugf("%v clients renewed", count)
	}

	needRestart1, count, err := s.disableInvalidClients(tx)
	if err != nil {
		logger.Warning("Error in disabling invalid clients:", err)
	} else if count > 0 {
		logger.Debugf("%v clients disabled", count)
	}

	needRestart2, count, err := s.disableInvalidInbounds(tx)
	if err != nil {
		logger.Warning("Error in disabling invalid inbounds:", err)
	} else if count > 0 {
		logger.Debugf("%v inbounds disabled", count)
	}
	return nil, (needRestart0 || needRestart1 || needRestart2)
}

func (s *InboundService) addInboundTraffic(tx *gorm.DB, traffics []*xray.Traffic) error {
	if len(traffics) == 0 {
		return nil
	}

	var err error

	for _, traffic := range traffics {
		if traffic.IsInbound {
			err = s.inbounds().AddInboundTrafficByTag(tx, traffic.Tag, traffic.Up, traffic.Down)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *InboundService) addClientTraffic(tx *gorm.DB, traffics []*xray.ClientTraffic) (err error) {
	if len(traffics) == 0 {
		// Empty onlineUsers
		if p != nil {
			p.SetOnlineClients(make([]string, 0))
		}
		return nil
	}

	onlineClients := make([]string, 0)

	emails := make([]string, 0, len(traffics))
	for _, traffic := range traffics {
		emails = append(emails, traffic.Email)
	}
	dbClientTraffics, err := s.inbounds().ListClientTrafficsByEmailsTx(tx, emails)
	if err != nil {
		return err
	}

	// Avoid empty slice error
	if len(dbClientTraffics) == 0 {
		return nil
	}

	dbClientTraffics, err = s.adjustTraffics(tx, dbClientTraffics)
	if err != nil {
		return err
	}

	for dbTraffic_index := range dbClientTraffics {
		for traffic_index := range traffics {
			if dbClientTraffics[dbTraffic_index].Email == traffics[traffic_index].Email {
				dbClientTraffics[dbTraffic_index].Up += traffics[traffic_index].Up
				dbClientTraffics[dbTraffic_index].Down += traffics[traffic_index].Down
				dbClientTraffics[dbTraffic_index].AllTime += (traffics[traffic_index].Up + traffics[traffic_index].Down)

				// Add user in onlineUsers array on traffic
				if traffics[traffic_index].Up+traffics[traffic_index].Down > 0 {
					onlineClients = append(onlineClients, traffics[traffic_index].Email)
					dbClientTraffics[dbTraffic_index].LastOnline = time.Now().UnixMilli()
				}
				break
			}
		}
	}

	// Set onlineUsers
	p.SetOnlineClients(onlineClients)

	err = s.inbounds().SaveClientTraffics(tx, dbClientTraffics)
	if err != nil {
		logger.Warning("AddClientTraffic update data ", err)
	}

	return nil
}

func (s *InboundService) adjustTraffics(tx *gorm.DB, dbClientTraffics []*xray.ClientTraffic) ([]*xray.ClientTraffic, error) {
	inboundIds := make([]int, 0, len(dbClientTraffics))
	for _, dbClientTraffic := range dbClientTraffics {
		if dbClientTraffic.ExpiryTime < 0 {
			inboundIds = append(inboundIds, dbClientTraffic.InboundId)
		}
	}

	if len(inboundIds) > 0 {
		inbounds, err := s.inbounds().ListInboundsByIDs(tx, inboundIds)
		if err != nil {
			return nil, err
		}
		for inbound_index := range inbounds {
			settings := map[string]any{}
			if err := json.Unmarshal([]byte(inbounds[inbound_index].Settings), &settings); err != nil {
				return nil, err
			}
			clients, ok := settings["clients"].([]any)
			if ok {
				var newClients []any
				for client_index := range clients {
					c := clients[client_index].(map[string]any)
					for traffic_index := range dbClientTraffics {
						if dbClientTraffics[traffic_index].ExpiryTime < 0 && c["email"] == dbClientTraffics[traffic_index].Email {
							oldExpiryTime := c["expiryTime"].(float64)
							newExpiryTime := (time.Now().Unix() * 1000) - int64(oldExpiryTime)
							c["expiryTime"] = newExpiryTime
							c["updated_at"] = time.Now().Unix() * 1000
							dbClientTraffics[traffic_index].ExpiryTime = newExpiryTime
							break
						}
					}
					// Backfill created_at and updated_at
					if _, ok := c["created_at"]; !ok {
						c["created_at"] = time.Now().Unix() * 1000
					}
					c["updated_at"] = time.Now().Unix() * 1000
					newClients = append(newClients, any(c))
				}
				settings["clients"] = newClients
				modifiedSettings, err := json.MarshalIndent(settings, "", "  ")
				if err != nil {
					return nil, err
				}

				inbounds[inbound_index].Settings = string(modifiedSettings)
			}
		}
		err = s.inbounds().SaveInbounds(tx, inbounds)
		if err != nil {
			logger.Warning("AddClientTraffic update inbounds ", err)
			logger.Error(inbounds)
		}
	}

	return dbClientTraffics, nil
}

func (s *InboundService) autoRenewClients(tx *gorm.DB) (bool, int64, error) {
	// check for time expired
	now := time.Now().Unix() * 1000
	var err, err1 error

	traffics, err := s.inbounds().ListRenewableClientTraffics(tx, now)
	if err != nil {
		return false, 0, err
	}
	// return if there is no client to renew
	if len(traffics) == 0 {
		return false, 0, nil
	}

	var inbound_ids []int
	var inbounds []*model.Inbound
	needRestart := false
	var clientsToAdd []struct {
		protocol string
		tag      string
		client   map[string]any
	}

	for _, traffic := range traffics {
		inbound_ids = append(inbound_ids, traffic.InboundId)
	}
	inbounds, err = s.inbounds().ListInboundsByIDs(tx, inbound_ids)
	if err != nil {
		return false, 0, err
	}
	for inbound_index := range inbounds {
		settings := map[string]any{}
		if err := json.Unmarshal([]byte(inbounds[inbound_index].Settings), &settings); err != nil {
			return false, 0, err
		}
		clients := settings["clients"].([]any)
		for client_index := range clients {
			c := clients[client_index].(map[string]any)
			for traffic_index, traffic := range traffics {
				if traffic.Email == c["email"].(string) {
					newExpiryTime := traffic.ExpiryTime
					for newExpiryTime < now {
						newExpiryTime += (int64(traffic.Reset) * 86400000)
					}
					c["expiryTime"] = newExpiryTime
					traffics[traffic_index].ExpiryTime = newExpiryTime
					traffics[traffic_index].Down = 0
					traffics[traffic_index].Up = 0
					if !traffic.Enable {
						traffics[traffic_index].Enable = true
						clientsToAdd = append(clientsToAdd,
							struct {
								protocol string
								tag      string
								client   map[string]any
							}{
								protocol: xrayRuntimeProtocol(inbounds[inbound_index].Protocol),
								tag:      inbounds[inbound_index].Tag,
								client:   c,
							})
					}
					clients[client_index] = any(c)
					break
				}
			}
		}
		settings["clients"] = clients
		newSettings, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			return false, 0, err
		}
		inbounds[inbound_index].Settings = string(newSettings)
	}
	err = s.inbounds().SaveInbounds(tx, inbounds)
	if err != nil {
		return false, 0, err
	}
	err = s.inbounds().SaveClientTraffics(tx, traffics)
	if err != nil {
		return false, 0, err
	}
	if p != nil {
		withXrayAPISyncLock(func() {
			err1 = s.xrayApi.Init(p.GetAPIPort())
			if err1 != nil {
				needRestart = true
				return
			}
			defer s.xrayApi.Close()
			for _, clientToAdd := range clientsToAdd {
				err1 = s.xrayApi.AddUser(clientToAdd.protocol, clientToAdd.tag, clientToAdd.client)
				if err1 != nil {
					needRestart = true
				}
			}
		})
	}
	return needRestart, int64(len(traffics)), nil
}

func (s *InboundService) disableInvalidInbounds(tx *gorm.DB) (bool, int64, error) {
	now := time.Now().Unix() * 1000
	needRestart := false

	if p != nil {
		tags, err := s.inbounds().ListInvalidInboundTags(tx, now)
		if err != nil {
			return false, 0, err
		}
		withXrayAPISyncLock(func() {
			if err1 := s.xrayApi.Init(p.GetAPIPort()); err1 != nil {
				logger.Debug("Unable to init xray api:", err1)
				needRestart = true
			} else {
				defer s.xrayApi.Close()
				for _, tag := range tags {
					err1 := s.xrayApi.DelInbound(tag)
					if err1 == nil {
						logger.Debug("Inbound disabled by api:", tag)
					} else {
						logger.Debug("Error in disabling inbound by api:", err1)
						needRestart = true
					}
				}
			}
		})
	}

	count, err := s.inbounds().DisableInvalidInbounds(tx, now)
	return needRestart, count, err
}

func (s *InboundService) disableInvalidClients(tx *gorm.DB) (bool, int64, error) {
	now := time.Now().Unix() * 1000
	needRestart := false

	if p != nil {
		results, err := s.inbounds().ListInvalidClientTrafficTargets(tx, now)
		if err != nil {
			return false, 0, err
		}
		withXrayAPISyncLock(func() {
			if err1 := s.xrayApi.Init(p.GetAPIPort()); err1 != nil {
				logger.Debug("Unable to init xray api:", err1)
				needRestart = true
			} else {
				defer s.xrayApi.Close()
				for _, result := range results {
					err1 := s.xrayApi.RemoveUser(result.Tag, result.Email)
					if err1 == nil {
						logger.Debug("Client disabled by api:", result.Email)
					} else if strings.Contains(err1.Error(), fmt.Sprintf("User %s not found.", result.Email)) {
						logger.Debug("User is already disabled. Nothing to do more...")
					} else {
						logger.Debug("Error in disabling client by api:", err1)
						needRestart = true
					}
				}
			}
		})
	}
	count, err := s.inbounds().DisableInvalidClientTraffics(tx, now)
	return needRestart, count, err
}

func (s *InboundService) GetInboundTags() (string, error) {
	inboundTags, err := s.inbounds().ListTags()
	if err != nil && err != gorm.ErrRecordNotFound {
		return "", err
	}
	tags, _ := json.Marshal(inboundTags)
	return string(tags), nil
}

func (s *InboundService) MigrationRemoveOrphanedTraffics() {
	_ = s.inbounds().DeleteOrphanedClientTraffics()
}

func (s *InboundService) AddClientStat(tx *gorm.DB, inboundId int, client *model.Client) error {
	clientTraffic := xray.ClientTraffic{}
	clientTraffic.InboundId = inboundId
	clientTraffic.Email = client.Email
	clientTraffic.Total = client.TotalGB
	clientTraffic.ExpiryTime = client.ExpiryTime
	clientTraffic.Enable = client.Enable
	clientTraffic.Up = 0
	clientTraffic.Down = 0
	clientTraffic.Reset = client.Reset
	return s.inbounds().CreateClientTraffic(tx, &clientTraffic)
}

func (s *InboundService) UpdateClientStat(tx *gorm.DB, email string, client *model.Client) error {
	return s.inbounds().UpdateClientTrafficByEmail(tx, email, client)
}

func (s *InboundService) UpdateClientIPs(tx *gorm.DB, oldEmail string, newEmail string) error {
	return s.inbounds().UpdateInboundClientIPs(tx, oldEmail, newEmail)
}

func (s *InboundService) DelClientStat(tx *gorm.DB, email string) error {
	return s.inbounds().DeleteClientTrafficByEmail(tx, email)
}

func (s *InboundService) DelClientIPs(tx *gorm.DB, email string) error {
	return s.inbounds().DeleteInboundClientIPsByEmail(tx, email)
}

func (s *InboundService) GetClientInboundByTrafficID(trafficId int) (traffic *xray.ClientTraffic, inbound *model.Inbound, err error) {
	traffic, err = s.inbounds().GetClientTrafficByID(trafficId)
	if err != nil {
		logger.Warningf("Error retrieving ClientTraffic with trafficId %d: %v", trafficId, err)
		return nil, nil, err
	}
	if traffic != nil {
		inbound, err = s.GetInbound(traffic.InboundId)
		return traffic, inbound, err
	}
	return nil, nil, nil
}

func (s *InboundService) GetClientInboundByEmail(email string) (traffic *xray.ClientTraffic, inbound *model.Inbound, err error) {
	traffic, err = s.inbounds().GetClientTrafficByEmail(email)
	if err != nil {
		logger.Warningf("Error retrieving ClientTraffic with email %s: %v", email, err)
		return nil, nil, err
	}
	if traffic != nil {
		inbound, err = s.GetInbound(traffic.InboundId)
		return traffic, inbound, err
	}
	return nil, nil, nil
}

func (s *InboundService) GetClientByEmail(clientEmail string) (*xray.ClientTraffic, *model.Client, error) {
	traffic, inbound, err := s.GetClientInboundByEmail(clientEmail)
	if err != nil {
		return nil, nil, err
	}
	if inbound == nil {
		return nil, nil, common.NewError("Inbound Not Found For Email:", clientEmail)
	}

	clients, err := s.GetClients(inbound)
	if err != nil {
		return nil, nil, err
	}

	for _, client := range clients {
		if client.Email == clientEmail {
			return traffic, &client, nil
		}
	}

	return nil, nil, common.NewError("Client Not Found In Inbound For Email:", clientEmail)
}

func (s *InboundService) SetClientTelegramUserID(trafficId int, tgId int64) (bool, error) {
	traffic, inbound, err := s.GetClientInboundByTrafficID(trafficId)
	if err != nil {
		return false, err
	}
	if inbound == nil {
		return false, common.NewError("Inbound Not Found For Traffic ID:", trafficId)
	}

	clientEmail := traffic.Email

	oldClients, err := s.GetClients(inbound)
	if err != nil {
		return false, err
	}

	clientId := s.getClientPrimaryKeyByEmail(inbound.Protocol, oldClients, clientEmail)

	if len(clientId) == 0 {
		return false, common.NewError("Client Not Found For Email:", clientEmail)
	}

	var settings map[string]any
	err = json.Unmarshal([]byte(inbound.Settings), &settings)
	if err != nil {
		return false, err
	}
	clients := settings["clients"].([]any)
	var newClients []any
	for client_index := range clients {
		c := clients[client_index].(map[string]any)
		if c["email"] == clientEmail {
			c["tgId"] = tgId
			c["updated_at"] = time.Now().Unix() * 1000
			newClients = append(newClients, any(c))
		}
	}
	settings["clients"] = newClients
	modifiedSettings, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return false, err
	}
	inbound.Settings = string(modifiedSettings)
	needRestart, err := s.UpdateInboundClient(inbound, clientId)
	return needRestart, err
}

func (s *InboundService) checkIsEnabledByEmail(clientEmail string) (bool, error) {
	_, inbound, err := s.GetClientInboundByEmail(clientEmail)
	if err != nil {
		return false, err
	}
	if inbound == nil {
		return false, common.NewError("Inbound Not Found For Email:", clientEmail)
	}

	clients, err := s.GetClients(inbound)
	if err != nil {
		return false, err
	}

	isEnable := false

	for _, client := range clients {
		if client.Email == clientEmail {
			isEnable = client.Enable
			break
		}
	}

	return isEnable, err
}

func (s *InboundService) ToggleClientEnableByEmail(clientEmail string) (bool, bool, error) {
	_, inbound, err := s.GetClientInboundByEmail(clientEmail)
	if err != nil {
		return false, false, err
	}
	if inbound == nil {
		return false, false, common.NewError("Inbound Not Found For Email:", clientEmail)
	}

	oldClients, err := s.GetClients(inbound)
	if err != nil {
		return false, false, err
	}

	clientId := s.getClientPrimaryKeyByEmail(inbound.Protocol, oldClients, clientEmail)
	clientOldEnabled := false

	for _, oldClient := range oldClients {
		if oldClient.Email == clientEmail {
			clientOldEnabled = oldClient.Enable
			break
		}
	}

	if len(clientId) == 0 {
		return false, false, common.NewError("Client Not Found For Email:", clientEmail)
	}

	var settings map[string]any
	err = json.Unmarshal([]byte(inbound.Settings), &settings)
	if err != nil {
		return false, false, err
	}
	clients := settings["clients"].([]any)
	var newClients []any
	for client_index := range clients {
		c := clients[client_index].(map[string]any)
		if c["email"] == clientEmail {
			c["enable"] = !clientOldEnabled
			c["updated_at"] = time.Now().Unix() * 1000
			newClients = append(newClients, any(c))
		}
	}
	settings["clients"] = newClients
	modifiedSettings, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return false, false, err
	}
	inbound.Settings = string(modifiedSettings)

	needRestart, err := s.UpdateInboundClient(inbound, clientId)
	if err != nil {
		return false, needRestart, err
	}

	return !clientOldEnabled, needRestart, nil
}

// SetClientEnableByEmail sets client enable state to desired value; returns (changed, needRestart, error)
func (s *InboundService) SetClientEnableByEmail(clientEmail string, enable bool) (bool, bool, error) {
	current, err := s.checkIsEnabledByEmail(clientEmail)
	if err != nil {
		return false, false, err
	}
	if current == enable {
		return false, false, nil
	}
	newEnabled, needRestart, err := s.ToggleClientEnableByEmail(clientEmail)
	if err != nil {
		return false, needRestart, err
	}
	return newEnabled == enable, needRestart, nil
}

func (s *InboundService) ResetClientIpLimitByEmail(clientEmail string, count int) (bool, error) {
	_, inbound, err := s.GetClientInboundByEmail(clientEmail)
	if err != nil {
		return false, err
	}
	if inbound == nil {
		return false, common.NewError("Inbound Not Found For Email:", clientEmail)
	}

	oldClients, err := s.GetClients(inbound)
	if err != nil {
		return false, err
	}

	clientId := s.getClientPrimaryKeyByEmail(inbound.Protocol, oldClients, clientEmail)

	if len(clientId) == 0 {
		return false, common.NewError("Client Not Found For Email:", clientEmail)
	}

	var settings map[string]any
	err = json.Unmarshal([]byte(inbound.Settings), &settings)
	if err != nil {
		return false, err
	}
	clients := settings["clients"].([]any)
	var newClients []any
	for client_index := range clients {
		c := clients[client_index].(map[string]any)
		if c["email"] == clientEmail {
			c["limitIp"] = count
			c["updated_at"] = time.Now().Unix() * 1000
			newClients = append(newClients, any(c))
		}
	}
	settings["clients"] = newClients
	modifiedSettings, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return false, err
	}
	inbound.Settings = string(modifiedSettings)
	needRestart, err := s.UpdateInboundClient(inbound, clientId)
	return needRestart, err
}

func (s *InboundService) ResetClientExpiryTimeByEmail(clientEmail string, expiry_time int64) (bool, error) {
	_, inbound, err := s.GetClientInboundByEmail(clientEmail)
	if err != nil {
		return false, err
	}
	if inbound == nil {
		return false, common.NewError("Inbound Not Found For Email:", clientEmail)
	}

	oldClients, err := s.GetClients(inbound)
	if err != nil {
		return false, err
	}

	clientId := s.getClientPrimaryKeyByEmail(inbound.Protocol, oldClients, clientEmail)

	if len(clientId) == 0 {
		return false, common.NewError("Client Not Found For Email:", clientEmail)
	}

	var settings map[string]any
	err = json.Unmarshal([]byte(inbound.Settings), &settings)
	if err != nil {
		return false, err
	}
	clients := settings["clients"].([]any)
	var newClients []any
	for client_index := range clients {
		c := clients[client_index].(map[string]any)
		if c["email"] == clientEmail {
			c["expiryTime"] = expiry_time
			c["updated_at"] = time.Now().Unix() * 1000
			newClients = append(newClients, any(c))
		}
	}
	settings["clients"] = newClients
	modifiedSettings, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return false, err
	}
	inbound.Settings = string(modifiedSettings)
	needRestart, err := s.UpdateInboundClient(inbound, clientId)
	return needRestart, err
}

func (s *InboundService) ResetClientTrafficLimitByEmail(clientEmail string, totalGB int) (bool, error) {
	if totalGB < 0 {
		return false, common.NewError("totalGB must be >= 0")
	}
	_, inbound, err := s.GetClientInboundByEmail(clientEmail)
	if err != nil {
		return false, err
	}
	if inbound == nil {
		return false, common.NewError("Inbound Not Found For Email:", clientEmail)
	}

	oldClients, err := s.GetClients(inbound)
	if err != nil {
		return false, err
	}

	clientId := s.getClientPrimaryKeyByEmail(inbound.Protocol, oldClients, clientEmail)

	if len(clientId) == 0 {
		return false, common.NewError("Client Not Found For Email:", clientEmail)
	}

	var settings map[string]any
	err = json.Unmarshal([]byte(inbound.Settings), &settings)
	if err != nil {
		return false, err
	}
	clients := settings["clients"].([]any)
	var newClients []any
	for client_index := range clients {
		c := clients[client_index].(map[string]any)
		if c["email"] == clientEmail {
			c["totalGB"] = totalGB * 1024 * 1024 * 1024
			c["updated_at"] = time.Now().Unix() * 1000
			newClients = append(newClients, any(c))
		}
	}
	settings["clients"] = newClients
	modifiedSettings, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return false, err
	}
	inbound.Settings = string(modifiedSettings)
	needRestart, err := s.UpdateInboundClient(inbound, clientId)
	return needRestart, err
}

func (s *InboundService) ResetClientTrafficByEmail(clientEmail string) error {
	return s.inbounds().ResetClientTrafficByEmail(clientEmail)
}

func (s *InboundService) ResetClientTraffic(id int, clientEmail string) (bool, error) {
	needRestart := false

	traffic, inbound, err := s.GetClientInboundByEmail(clientEmail)
	if err != nil {
		return false, err
	}
	if traffic == nil || inbound == nil {
		return false, common.NewError("Inbound Not Found For Email:", clientEmail)
	}

	if !traffic.Enable {
		clients, err := s.GetClients(inbound)
		if err != nil {
			return false, err
		}
		for _, client := range clients {
			if client.Email == clientEmail && client.Enable {
				withXrayAPISyncLock(func() {
					if err1 := s.xrayApi.Init(p.GetAPIPort()); err1 != nil {
						logger.Debug("Unable to init xray api:", err1)
						needRestart = true
					} else {
						defer s.xrayApi.Close()
						cipher := ""
						if string(inbound.Protocol) == "shadowsocks" {
							var oldSettings map[string]any
							err = json.Unmarshal([]byte(inbound.Settings), &oldSettings)
							if err != nil {
								needRestart = true
								return
							}
							cipher, _ = oldSettings["method"].(string)
						}
						err1 := s.xrayApi.AddUser(xrayRuntimeProtocol(inbound.Protocol), inbound.Tag, map[string]any{
							"email":    client.Email,
							"id":       client.ID,
							"auth":     client.Auth,
							"security": client.Security,
							"flow":     client.Flow,
							"password": client.Password,
							"cipher":   cipher,
						})
						if err1 == nil {
							logger.Debug("Client enabled due to reset traffic:", clientEmail)
						} else {
							logger.Debug("Error in enabling client by api:", err1)
							needRestart = true
						}
					}
				})
				if err != nil {
					return false, err
				}
				break
			}
		}
	}

	traffic.Up = 0
	traffic.Down = 0
	traffic.Enable = true

	if err := s.inbounds().SaveClientTraffic(traffic); err != nil {
		return false, err
	}

	return needRestart, nil
}

func (s *InboundService) ResetAllClientTraffics(id int) error {
	return s.inbounds().ResetAllClientTraffics(id)
}

func (s *InboundService) ResetAllTraffics() error {
	return s.inbounds().ResetAllTraffics()
}

func (s *InboundService) ResetInboundTraffic(id int) error {
	return s.inbounds().ResetInboundTraffic(id)
}

func (s *InboundService) DelDepletedClients(id int) (err error) {
	db := database.GetDB()
	tx := db.Begin()
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	// Only consider truly depleted clients: expired OR traffic exhausted
	now := time.Now().Unix() * 1000
	depletedClients, err := s.inbounds().ListDepletedClientGroups(tx, id, now)
	if err != nil {
		return err
	}

	for _, depletedClient := range depletedClients {
		oldInbound, err := s.GetInbound(depletedClient.InboundID)
		if err != nil {
			return err
		}
		var oldSettings map[string]any
		err = json.Unmarshal([]byte(oldInbound.Settings), &oldSettings)
		if err != nil {
			return err
		}

		oldClients := oldSettings["clients"].([]any)
		var newClients []any
		for _, client := range oldClients {
			deplete := false
			c := client.(map[string]any)
			for _, email := range depletedClient.Emails {
				if email == c["email"].(string) {
					deplete = true
					break
				}
			}
			if !deplete {
				newClients = append(newClients, client)
			}
		}
		if len(newClients) > 0 {
			oldSettings["clients"] = newClients

			newSettings, err := json.MarshalIndent(oldSettings, "", "  ")
			if err != nil {
				return err
			}

			oldInbound.Settings = string(newSettings)
			err = s.inbounds().SaveInbound(tx, oldInbound)
			if err != nil {
				return err
			}
		} else {
			// Delete inbound if no client remains
			if _, err := s.DelInbound(depletedClient.InboundID); err != nil {
				return err
			}
		}
	}

	// Delete stats only for truly depleted clients
	err = s.inbounds().DeleteDepletedClientTraffics(tx, id, now)
	if err != nil {
		return err
	}

	return nil
}

func (s *InboundService) GetClientTrafficTgBot(tgId int64) ([]*xray.ClientTraffic, error) {
	inbounds, err := s.GetAllInbounds()
	if err != nil {
		return nil, err
	}

	var emails []string
	clientByEmail := make(map[string]model.Client)
	for _, inbound := range inbounds {
		clients, err := s.GetClients(inbound)
		if err != nil {
			logger.Errorf("Error retrieving clients for inbound %d: %v", inbound.Id, err)
			continue
		}
		for _, client := range clients {
			if client.TgID == tgId && client.Email != "" {
				clientByEmail[strings.ToLower(client.Email)] = client
				emails = append(emails, client.Email)
			}
		}
	}

	if len(emails) == 0 {
		return nil, nil
	}

	traffics, err := s.inbounds().ListClientTrafficsByEmails(emails)
	if err != nil {
		logger.Errorf("Error retrieving ClientTraffic for emails %v: %v", emails, err)
		return nil, err
	}

	result := make([]*xray.ClientTraffic, 0, len(traffics))
	// Populate UUID and other client data for each traffic record
	for i := range traffics {
		traffic := &traffics[i]
		if client, ok := clientByEmail[strings.ToLower(traffic.Email)]; ok {
			traffic.Enable = client.Enable
			traffic.UUID = client.ID
			traffic.SubId = client.SubID
		}
		result = append(result, traffic)
	}

	return result, nil
}

func (s *InboundService) GetClientTrafficByEmail(email string) (traffic *xray.ClientTraffic, err error) {
	// Prefer retrieving along with client to reflect actual enabled state from inbound settings
	t, client, err := s.GetClientByEmail(email)
	if err != nil {
		logger.Warningf("Error retrieving ClientTraffic with email %s: %v", email, err)
		return nil, err
	}
	if t != nil && client != nil {
		t.UUID = client.ID
		t.SubId = client.SubID
		return t, nil
	}
	return nil, nil
}

func (s *InboundService) UpdateClientTrafficByEmail(email string, upload int64, download int64) error {
	err := s.inbounds().UpdateClientTrafficUsageByEmail(email, upload, download)
	if err != nil {
		logger.Warningf("Error updating ClientTraffic with email %s: %v", email, err)
		return err
	}
	return nil
}

func (s *InboundService) GetClientTrafficByID(id string) ([]xray.ClientTraffic, error) {
	traffics, err := s.inbounds().ListClientTrafficsByClientID(id)
	if err != nil {
		logger.Debug(err)
		return nil, err
	}
	// Reconcile enable flag with client settings per email to avoid stale DB value
	for i := range traffics {
		if ct, client, e := s.GetClientByEmail(traffics[i].Email); e == nil && ct != nil && client != nil {
			traffics[i].Enable = client.Enable
			traffics[i].UUID = client.ID
			traffics[i].SubId = client.SubID
		}
	}
	return traffics, err
}

func (s *InboundService) SearchClientTraffic(query string) (traffic *xray.ClientTraffic, err error) {
	inbound, err := s.inbounds().FindInboundBySettingsContains(query)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warningf("Inbound settings containing query %s not found: %v", query, err)
			return nil, err
		}
		logger.Errorf("Error searching for inbound settings with query %s: %v", query, err)
		return nil, err
	}
	traffic = &xray.ClientTraffic{}

	traffic.InboundId = inbound.Id

	clients, err := parseInboundClients(inbound.Settings)
	if err != nil {
		logger.Errorf("Error unmarshalling inbound settings for inbound ID %d: %v", inbound.Id, err)
		return nil, err
	}

	for _, client := range clients {
		if (client.ID == query || client.Password == query) && client.Email != "" {
			traffic.Email = client.Email
			break
		}
	}

	if traffic.Email == "" {
		logger.Warningf("No client found with query %s in inbound ID %d", query, inbound.Id)
		return nil, gorm.ErrRecordNotFound
	}

	traffic, err = s.inbounds().GetClientTrafficByEmail(traffic.Email)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warningf("ClientTraffic for email %s not found: %v", traffic.Email, err)
			return nil, err
		}
		logger.Errorf("Error retrieving ClientTraffic for email %s: %v", traffic.Email, err)
		return nil, err
	}

	return traffic, nil
}

func (s *InboundService) GetInboundClientIps(clientEmail string) (string, error) {
	InboundClientIps, err := s.inbounds().FindInboundClientIps(clientEmail)
	if err != nil {
		return "", err
	}

	if InboundClientIps.Ips == "" {
		return "", nil
	}

	// Try to parse as new format (with timestamps)
	type IPWithTimestamp struct {
		IP        string `json:"ip"`
		Timestamp int64  `json:"timestamp"`
	}

	var ipsWithTime []IPWithTimestamp
	err = json.Unmarshal([]byte(InboundClientIps.Ips), &ipsWithTime)

	// If successfully parsed as new format, return with timestamps
	if err == nil && len(ipsWithTime) > 0 {
		return InboundClientIps.Ips, nil
	}

	// Otherwise, assume it's old format (simple string array)
	// Try to parse as simple array and convert to new format
	var oldIps []string
	err = json.Unmarshal([]byte(InboundClientIps.Ips), &oldIps)
	if err == nil && len(oldIps) > 0 {
		// Convert old format to new format with current timestamp
		newIpsWithTime := make([]IPWithTimestamp, len(oldIps))
		for i, ip := range oldIps {
			newIpsWithTime[i] = IPWithTimestamp{
				IP:        ip,
				Timestamp: time.Now().Unix(),
			}
		}
		result, _ := json.Marshal(newIpsWithTime)
		return string(result), nil
	}

	// Return as-is if parsing fails
	return InboundClientIps.Ips, nil
}

func (s *InboundService) ClearClientIps(clientEmail string) error {
	err := s.inbounds().ClearInboundClientIps(clientEmail)
	if err != nil {
		return err
	}
	return nil
}

func (s *InboundService) SearchInbounds(query string) ([]*model.Inbound, error) {
	return s.inbounds().SearchByRemark(query)
}

func (s *InboundService) MigrationRequirements() {
	db := database.GetDB()
	tx := db.Begin()
	var err error
	defer func() {
		if err == nil {
			tx.Commit()
			if dbErr := db.Exec(`VACUUM "main"`).Error; dbErr != nil {
				logger.Warningf("VACUUM failed: %v", dbErr)
			}
		} else {
			tx.Rollback()
		}
	}()

	// Calculate and backfill all_time from up+down for inbounds and clients
	err = tx.Exec(`
		UPDATE inbounds
		SET all_time = IFNULL(up, 0) + IFNULL(down, 0)
		WHERE IFNULL(all_time, 0) = 0 AND (IFNULL(up, 0) + IFNULL(down, 0)) > 0
	`).Error
	if err != nil {
		return
	}
	err = tx.Exec(`
		UPDATE client_traffics
		SET all_time = IFNULL(up, 0) + IFNULL(down, 0)
		WHERE IFNULL(all_time, 0) = 0 AND (IFNULL(up, 0) + IFNULL(down, 0)) > 0
	`).Error

	if err != nil {
		return
	}

	// Fix inbounds based problems
	var inbounds []*model.Inbound
	err = tx.Model(model.Inbound{}).Where("protocol IN (?)", []string{"vmess", "vless", "trojan"}).Find(&inbounds).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return
	}
	for inbound_index := range inbounds {
		settings := map[string]any{}
		err = json.Unmarshal([]byte(inbounds[inbound_index].Settings), &settings)
		if err != nil {
			return
		}
		clients, ok := settings["clients"].([]any)
		if ok {
			// Fix Client configuration problems
			var newClients []any
			for client_index := range clients {
				c := clients[client_index].(map[string]any)

				// Add email='' if it is not exists
				if _, ok := c["email"]; !ok {
					c["email"] = ""
				}

				// Convert string tgId to int64
				if _, ok := c["tgId"]; ok {
					var tgId any = c["tgId"]
					if tgIdStr, ok2 := tgId.(string); ok2 {
						tgIdInt64, err := strconv.ParseInt(strings.ReplaceAll(tgIdStr, " ", ""), 10, 64)
						if err == nil {
							c["tgId"] = tgIdInt64
						}
					}
				}

				// Remove "flow": "xtls-rprx-direct"
				if _, ok := c["flow"]; ok {
					if c["flow"] == "xtls-rprx-direct" {
						c["flow"] = ""
					}
				}
				// Backfill created_at and updated_at
				if _, ok := c["created_at"]; !ok {
					c["created_at"] = time.Now().Unix() * 1000
				}
				c["updated_at"] = time.Now().Unix() * 1000
				newClients = append(newClients, any(c))
			}
			settings["clients"] = newClients
			modifiedSettings, err := json.MarshalIndent(settings, "", "  ")
			if err != nil {
				return
			}

			inbounds[inbound_index].Settings = string(modifiedSettings)
		}

		// Add client traffic row for all clients which has email
		modelClients, err := s.GetClients(inbounds[inbound_index])
		if err != nil {
			return
		}
		for _, modelClient := range modelClients {
			if len(modelClient.Email) > 0 {
				var count int64
				err = tx.Model(xray.ClientTraffic{}).Where("email = ?", modelClient.Email).Count(&count).Error
				if err != nil {
					return
				}
				if count == 0 {
					err = s.AddClientStat(tx, inbounds[inbound_index].Id, &modelClient)
					if err != nil {
						return
					}
				}
			}
		}
	}
	err = s.inbounds().SaveInbounds(tx, inbounds)
	if err != nil {
		return
	}

	// Remove orphaned traffics
	err = tx.Where("inbound_id = 0").Delete(xray.ClientTraffic{}).Error
	if err != nil {
		return
	}

	// Migrate old MultiDomain to External Proxy
	var externalProxy []struct {
		Id             int
		Port           int
		StreamSettings []byte
	}
	err = tx.Raw(`select id, port, stream_settings
	from inbounds
	WHERE protocol in ('vmess','vless','trojan')
	  AND json_extract(stream_settings, '$.security') = 'tls'
	  AND json_extract(stream_settings, '$.tlsSettings.settings.domains') IS NOT NULL`).Scan(&externalProxy).Error
	if err != nil || len(externalProxy) == 0 {
		return
	}

	for _, ep := range externalProxy {
		var reverses any
		var stream map[string]any
		err = json.Unmarshal(ep.StreamSettings, &stream)
		if err != nil {
			return
		}
		if tlsSettings, ok := stream["tlsSettings"].(map[string]any); ok {
			if settings, ok := tlsSettings["settings"].(map[string]any); ok {
				if domains, ok := settings["domains"].([]any); ok {
					for _, domain := range domains {
						if domainMap, ok := domain.(map[string]any); ok {
							domainMap["forceTls"] = "same"
							domainMap["port"] = ep.Port
							domainMap["dest"] = domainMap["domain"].(string)
							delete(domainMap, "domain")
						}
					}
				}
				reverses = settings["domains"]
				delete(settings, "domains")
			}
		}
		stream["externalProxy"] = reverses
		newStream, err := json.MarshalIndent(stream, " ", "  ")
		if err != nil {
			return
		}
		err = tx.Model(model.Inbound{}).Where("id = ?", ep.Id).Update("stream_settings", newStream).Error
		if err != nil {
			return
		}
	}

	err = tx.Raw(`UPDATE inbounds
	SET tag = REPLACE(tag, '0.0.0.0:', '')
	WHERE INSTR(tag, '0.0.0.0:') > 0;`).Error
	if err != nil {
		return
	}
}

func (s *InboundService) MigrateDB() {
	s.MigrationRequirements()
	s.MigrationRemoveOrphanedTraffics()
}

func (s *InboundService) GetOnlineClients() []string {
	return p.GetOnlineClients()
}

func (s *InboundService) GetClientsLastOnline() (map[string]int64, error) {
	rows, err := s.inbounds().ListClientTrafficsLastOnline()
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(rows))
	for _, r := range rows {
		result[r.Email] = r.LastOnline
	}
	return result, nil
}

func (s *InboundService) FilterAndSortClientEmails(emails []string) ([]string, []string, error) {
	clients, err := s.inbounds().ListClientTrafficsByEmails(emails)
	if err != nil {
		return nil, nil, err
	}

	// Step 2: Sort clients by (Up + Down) descending
	sort.Slice(clients, func(i, j int) bool {
		return (clients[i].Up + clients[i].Down) > (clients[j].Up + clients[j].Down)
	})

	// Step 3: Extract sorted valid emails and track found ones
	validEmails := make([]string, 0, len(clients))
	found := make(map[string]bool)
	for _, client := range clients {
		validEmails = append(validEmails, client.Email)
		found[client.Email] = true
	}

	// Step 4: Identify emails that were not found in the database
	extraEmails := make([]string, 0)
	for _, email := range emails {
		if !found[email] {
			extraEmails = append(extraEmails, email)
		}
	}

	return validEmails, extraEmails, nil
}
func (s *InboundService) DelInboundClientByEmail(inboundId int, email string) (bool, error) {
	oldInbound, err := s.GetInbound(inboundId)
	if err != nil {
		logger.Error("Load Old Data Error")
		return false, err
	}

	var settings map[string]any
	if err := json.Unmarshal([]byte(oldInbound.Settings), &settings); err != nil {
		return false, err
	}

	interfaceClients, ok := settings["clients"].([]any)
	if !ok {
		return false, common.NewError("invalid clients format in inbound settings")
	}

	var newClients []any
	needApiDel := false
	found := false

	for _, client := range interfaceClients {
		c, ok := client.(map[string]any)
		if !ok {
			continue
		}
		if cEmail, ok := c["email"].(string); ok && cEmail == email {
			// matched client, drop it
			found = true
			needApiDel, _ = c["enable"].(bool)
		} else {
			newClients = append(newClients, client)
		}
	}

	if !found {
		return false, common.NewError(fmt.Sprintf("client with email %s not found", email))
	}
	if len(newClients) == 0 {
		return false, common.NewError("no client remained in Inbound")
	}

	settings["clients"] = newClients
	newSettings, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return false, err
	}

	oldInbound.Settings = string(newSettings)

	db := database.GetDB()

	// remove IP bindings
	if err := s.DelClientIPs(db, email); err != nil {
		logger.Error("Error in delete client IPs")
		return false, err
	}

	needRestart := false

	// remove stats too
	if len(email) > 0 {
		traffic, err := s.GetClientTrafficByEmail(email)
		if err != nil {
			return false, err
		}
		if traffic != nil {
			if err := s.DelClientStat(db, email); err != nil {
				logger.Error("Delete stats Data Error")
				return false, err
			}
		}

		if needApiDel {
			withXrayAPISyncLock(func() {
				if err1 := s.xrayApi.Init(p.GetAPIPort()); err1 != nil {
					logger.Debug("Unable to init xray api:", err1)
					needRestart = true
				} else {
					if err1 := s.xrayApi.RemoveUser(oldInbound.Tag, email); err1 == nil {
						logger.Debug("Client deleted by api:", email)
						needRestart = false
					} else {
						if strings.Contains(err1.Error(), fmt.Sprintf("User %s not found.", email)) {
							logger.Debug("User is already deleted. Nothing to do more...")
						} else {
							logger.Debug("Error in deleting client by api:", err1)
							needRestart = true
						}
					}
					s.xrayApi.Close()
				}
			})
		}
	}

	return needRestart, s.inbounds().SaveInbound(db, oldInbound)
}
