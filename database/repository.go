package database

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/xray"
	"gorm.io/gorm"
)

// Repositories groups typed data access boundaries for service-layer migration.
type Repositories struct {
	Users    UserRepository
	Settings SettingRepository
	Inbounds InboundRepository
}

// NewRepositories creates GORM-backed repositories for the current schema.
func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		Users:    &GormUserRepository{db: db},
		Settings: &GormSettingRepository{db: db},
		Inbounds: &GormInboundRepository{db: db},
	}
}

type UserRepository interface {
	First() (*model.User, error)
	FindByUsername(username string) (*model.User, error)
	Save(user *model.User) error
	UpdateCredentials(id int, username string, password string) error
}

type SettingRepository interface {
	Get(key string) (*model.Setting, error)
	Save(key string, value string) error
	AllExcept(keys ...string) ([]*model.Setting, error)
	DeleteAll() error
}

type InboundRepository interface {
	Get(id int) (*model.Inbound, error)
	ListByUserID(userID int) ([]*model.Inbound, error)
	ListAll() ([]*model.Inbound, error)
	ListOptionsByUserID(userID int) ([]InboundOptionRecord, error)
	ListByTrafficReset(period string) ([]*model.Inbound, error)
	PortExists(listen string, port int, ignoreID int) (bool, error)
	ListClientEmails() ([]string, error)
	GetClientTrafficByID(id int) (*xray.ClientTraffic, error)
	GetClientTrafficByEmail(email string) (*xray.ClientTraffic, error)
	ListClientTrafficsLastOnline() ([]xray.ClientTraffic, error)
	ListClientTrafficsByEmails(emails []string) ([]xray.ClientTraffic, error)
	ListClientTrafficsByEmailsTx(tx *gorm.DB, emails []string) ([]*xray.ClientTraffic, error)
	ListClientTrafficsByClientID(id string) ([]xray.ClientTraffic, error)
	ListInboundsByIDs(tx *gorm.DB, ids []int) ([]*model.Inbound, error)
	ListClientTrafficEnableByInboundID(tx *gorm.DB, inboundID int) ([]xray.ClientTraffic, error)
	IsClientTrafficEnabledByEmail(tx *gorm.DB, email string) (bool, error)
	ListDepletedClientGroups(tx *gorm.DB, inboundID int, now int64) ([]DepletedClientGroup, error)
	DeleteDepletedClientTraffics(tx *gorm.DB, inboundID int, now int64) error
	ListRenewableClientTraffics(tx *gorm.DB, now int64) ([]*xray.ClientTraffic, error)
	ListInvalidInboundTags(tx *gorm.DB, now int64) ([]string, error)
	DisableInvalidInbounds(tx *gorm.DB, now int64) (int64, error)
	ListInvalidClientTrafficTargets(tx *gorm.DB, now int64) ([]ClientTrafficTarget, error)
	DisableInvalidClientTraffics(tx *gorm.DB, now int64) (int64, error)
	FindInboundBySettingsContains(query string) (*model.Inbound, error)
	SearchByRemark(query string) ([]*model.Inbound, error)
	ListTags() ([]string, error)
	DeleteOrphanedClientTraffics() error
	AddInboundTrafficByTag(tx *gorm.DB, tag string, upload int64, download int64) error
	UpdateClientTrafficUsageByEmail(email string, upload int64, download int64) error
	ResetClientTrafficByEmail(email string) error
	SaveClientTraffic(traffic *xray.ClientTraffic) error
	SaveClientTraffics(tx *gorm.DB, traffics []*xray.ClientTraffic) error
	ResetAllClientTraffics(id int) error
	ResetAllTraffics() error
	ResetInboundTraffic(id int) error
	CreateClientTraffic(tx *gorm.DB, clientTraffic *xray.ClientTraffic) error
	UpdateClientTrafficByEmail(tx *gorm.DB, email string, client *model.Client) error
	UpdateInboundClientIPs(tx *gorm.DB, oldEmail string, newEmail string) error
	FindInboundClientIps(clientEmail string) (*model.InboundClientIps, error)
	ClearInboundClientIps(clientEmail string) error
	DeleteClientTrafficByEmail(tx *gorm.DB, email string) error
	DeleteClientTrafficsByInboundID(tx *gorm.DB, inboundID int) error
	DeleteInboundClientIPsByEmail(tx *gorm.DB, email string) error
	DeleteByID(tx *gorm.DB, id int) error
	SaveInbound(tx *gorm.DB, inbound *model.Inbound) error
	SaveInbounds(tx *gorm.DB, inbounds []*model.Inbound) error
	Save(inbound *model.Inbound) error
}

type InboundOptionRecord struct {
	ID             int
	Remark         string
	Tag            string
	Protocol       string
	Port           int
	TLSFlowCapable bool
	SSMethod       string
}

type ClientTrafficTarget struct {
	Tag   string
	Email string
}

type DepletedClientGroup struct {
	InboundID int
	Emails    []string
}

type GormUserRepository struct {
	db *gorm.DB
}

func (r *GormUserRepository) First() (*model.User, error) {
	user := new(model.User)
	if err := r.db.Model(model.User{}).First(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (r *GormUserRepository) FindByUsername(username string) (*model.User, error) {
	user := new(model.User)
	if err := r.db.Model(model.User{}).Where("username = ?", username).First(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (r *GormUserRepository) Save(user *model.User) error {
	return r.db.Save(user).Error
}

func (r *GormUserRepository) UpdateCredentials(id int, username string, password string) error {
	return r.db.Model(model.User{}).
		Where("id = ?", id).
		Updates(map[string]any{"username": username, "password": password}).
		Error
}

type GormSettingRepository struct {
	db *gorm.DB
}

func (r *GormSettingRepository) Get(key string) (*model.Setting, error) {
	setting := new(model.Setting)
	if err := r.db.Model(model.Setting{}).Where("key = ?", key).First(setting).Error; err != nil {
		return nil, err
	}
	return setting, nil
}

func (r *GormSettingRepository) Save(key string, value string) error {
	setting, err := r.Get(key)
	if err != nil {
		if IsNotFound(err) {
			return r.db.Create(&model.Setting{
				Key:   key,
				Value: value,
			}).Error
		}
		return err
	}

	setting.Value = value
	return r.db.Save(setting).Error
}

func (r *GormSettingRepository) AllExcept(keys ...string) ([]*model.Setting, error) {
	settings := make([]*model.Setting, 0)
	query := r.db.Model(model.Setting{})
	if len(keys) > 0 {
		query = query.Not("key IN ?", keys)
	}
	if err := query.Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

func (r *GormSettingRepository) DeleteAll() error {
	return r.db.Where("1 = 1").Delete(&model.Setting{}).Error
}

type GormInboundRepository struct {
	db *gorm.DB
}

func (r *GormInboundRepository) Get(id int) (*model.Inbound, error) {
	inbound := new(model.Inbound)
	if err := r.db.First(inbound, id).Error; err != nil {
		return nil, err
	}
	return inbound, nil
}

func (r *GormInboundRepository) ListByUserID(userID int) ([]*model.Inbound, error) {
	inbounds := make([]*model.Inbound, 0)
	if err := r.db.Preload("ClientStats").Where("user_id = ?", userID).Find(&inbounds).Error; err != nil {
		return nil, err
	}
	return inbounds, nil
}

func (r *GormInboundRepository) ListAll() ([]*model.Inbound, error) {
	inbounds := make([]*model.Inbound, 0)
	if err := r.db.Preload("ClientStats").Find(&inbounds).Error; err != nil {
		return nil, err
	}
	return inbounds, nil
}

func (r *GormInboundRepository) ListOptionsByUserID(userID int) ([]InboundOptionRecord, error) {
	var rows []struct {
		ID             int    `gorm:"column:id"`
		Remark         string `gorm:"column:remark"`
		Tag            string `gorm:"column:tag"`
		Protocol       string `gorm:"column:protocol"`
		Port           int    `gorm:"column:port"`
		StreamSettings string `gorm:"column:stream_settings"`
		Settings       string `gorm:"column:settings"`
	}
	if err := r.db.Table("inbounds").
		Select("id, remark, tag, protocol, port, stream_settings, settings").
		Where("user_id = ?", userID).
		Order("id ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	options := make([]InboundOptionRecord, 0, len(rows))
	for _, row := range rows {
		options = append(options, InboundOptionRecord{
			ID:             row.ID,
			Remark:         row.Remark,
			Tag:            row.Tag,
			Protocol:       row.Protocol,
			Port:           row.Port,
			TLSFlowCapable: inboundCanEnableTLSFlow(row.Protocol, row.StreamSettings),
			SSMethod:       inboundShadowsocksMethod(row.Protocol, row.Settings),
		})
	}
	return options, nil
}

func (r *GormInboundRepository) ListByTrafficReset(period string) ([]*model.Inbound, error) {
	inbounds := make([]*model.Inbound, 0)
	if err := r.db.Model(model.Inbound{}).Where("traffic_reset = ?", period).Find(&inbounds).Error; err != nil {
		return nil, err
	}
	return inbounds, nil
}

func (r *GormInboundRepository) PortExists(listen string, port int, ignoreID int) (bool, error) {
	query := r.db.Model(model.Inbound{})
	if listen == "" || listen == "0.0.0.0" || listen == "::" || listen == "::0" {
		query = query.Where("port = ?", port)
	} else {
		query = query.Where("port = ?", port).
			Where(
				r.db.Model(model.Inbound{}).Where(
					"listen = ?", listen,
				).Or(
					"listen = \"\"",
				).Or(
					"listen = \"0.0.0.0\"",
				).Or(
					"listen = \"::\"",
				).Or(
					"listen = \"::0\""))
	}
	if ignoreID > 0 {
		query = query.Where("id != ?", ignoreID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *GormInboundRepository) ListClientEmails() ([]string, error) {
	emails := make([]string, 0)
	if err := r.db.Raw(`
		SELECT JSON_EXTRACT(client.value, '$.email')
		FROM inbounds,
			JSON_EACH(JSON_EXTRACT(inbounds.settings, '$.clients')) AS client
		`).Scan(&emails).Error; err != nil {
		return nil, err
	}
	return emails, nil
}

func (r *GormInboundRepository) GetClientTrafficByID(id int) (*xray.ClientTraffic, error) {
	traffic := new(xray.ClientTraffic)
	result := r.db.Model(xray.ClientTraffic{}).Where("id = ?", id).Find(traffic)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return traffic, nil
}

func (r *GormInboundRepository) GetClientTrafficByEmail(email string) (*xray.ClientTraffic, error) {
	traffic := new(xray.ClientTraffic)
	result := r.db.Model(xray.ClientTraffic{}).Where("email = ?", email).Find(traffic)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return traffic, nil
}

func (r *GormInboundRepository) ListClientTrafficsLastOnline() ([]xray.ClientTraffic, error) {
	rows := make([]xray.ClientTraffic, 0)
	if err := r.db.Model(xray.ClientTraffic{}).Select("email, last_online").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *GormInboundRepository) ListClientTrafficsByEmails(emails []string) ([]xray.ClientTraffic, error) {
	rows := make([]xray.ClientTraffic, 0)
	if err := r.db.Model(xray.ClientTraffic{}).Where("email IN ?", emails).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *GormInboundRepository) ListClientTrafficsByEmailsTx(tx *gorm.DB, emails []string) ([]*xray.ClientTraffic, error) {
	rows := make([]*xray.ClientTraffic, 0)
	if len(emails) == 0 {
		return rows, nil
	}
	if tx == nil {
		tx = r.db
	}
	if err := tx.Model(xray.ClientTraffic{}).Where("email IN (?)", emails).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *GormInboundRepository) ListClientTrafficsByClientID(id string) ([]xray.ClientTraffic, error) {
	traffics := make([]xray.ClientTraffic, 0)
	if err := r.db.Model(xray.ClientTraffic{}).Where(`email IN(
		SELECT JSON_EXTRACT(client.value, '$.email') as email
		FROM inbounds,
			JSON_EACH(JSON_EXTRACT(inbounds.settings, '$.clients')) AS client
		WHERE
			JSON_EXTRACT(client.value, '$.id') in (?)
		)`, id).Find(&traffics).Error; err != nil {
		return nil, err
	}
	return traffics, nil
}

func (r *GormInboundRepository) ListInboundsByIDs(tx *gorm.DB, ids []int) ([]*model.Inbound, error) {
	inbounds := make([]*model.Inbound, 0)
	if len(ids) == 0 {
		return inbounds, nil
	}
	if tx == nil {
		tx = r.db
	}
	if err := tx.Model(model.Inbound{}).Where("id IN (?)", ids).Find(&inbounds).Error; err != nil {
		return nil, err
	}
	return inbounds, nil
}

func (r *GormInboundRepository) ListClientTrafficEnableByInboundID(tx *gorm.DB, inboundID int) ([]xray.ClientTraffic, error) {
	rows := make([]xray.ClientTraffic, 0)
	if tx == nil {
		tx = r.db
	}
	if err := tx.Model(xray.ClientTraffic{}).
		Where("inbound_id = ?", inboundID).
		Select("email", "enable").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *GormInboundRepository) IsClientTrafficEnabledByEmail(tx *gorm.DB, email string) (bool, error) {
	if tx == nil {
		tx = r.db
	}
	var row struct {
		Enable bool
	}
	if err := tx.Model(xray.ClientTraffic{}).Select("enable").Where("email = ?", email).First(&row).Error; err != nil {
		return false, err
	}
	return row.Enable, nil
}

func (r *GormInboundRepository) ListDepletedClientGroups(tx *gorm.DB, inboundID int, now int64) ([]DepletedClientGroup, error) {
	if tx == nil {
		tx = r.db
	}

	whereText := "reset = 0 and inbound_id "
	if inboundID < 0 {
		whereText += "> ?"
	} else {
		whereText += "= ?"
	}

	rows := make([]struct {
		InboundID int    `gorm:"column:inbound_id"`
		Emails    string `gorm:"column:email"`
	}, 0)
	if err := tx.Model(xray.ClientTraffic{}).
		Where(whereText+" and ((total > 0 and up + down >= total) or (expiry_time > 0 and expiry_time <= ?))", inboundID, now).
		Select("inbound_id, GROUP_CONCAT(email) as email").
		Group("inbound_id").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	groups := make([]DepletedClientGroup, 0, len(rows))
	for _, row := range rows {
		emails := make([]string, 0)
		for _, email := range strings.Split(row.Emails, ",") {
			if email != "" {
				emails = append(emails, email)
			}
		}
		groups = append(groups, DepletedClientGroup{
			InboundID: row.InboundID,
			Emails:    emails,
		})
	}
	return groups, nil
}

func (r *GormInboundRepository) DeleteDepletedClientTraffics(tx *gorm.DB, inboundID int, now int64) error {
	if tx == nil {
		tx = r.db
	}
	whereText := "reset = 0 and inbound_id "
	if inboundID < 0 {
		whereText += "> ?"
	} else {
		whereText += "= ?"
	}
	return tx.Where(whereText+" and ((total > 0 and up + down >= total) or (expiry_time > 0 and expiry_time <= ?))", inboundID, now).Delete(xray.ClientTraffic{}).Error
}

func (r *GormInboundRepository) ListRenewableClientTraffics(tx *gorm.DB, now int64) ([]*xray.ClientTraffic, error) {
	traffics := make([]*xray.ClientTraffic, 0)
	if tx == nil {
		tx = r.db
	}
	if err := tx.Model(xray.ClientTraffic{}).Where("reset > 0 and expiry_time > 0 and expiry_time <= ?", now).Find(&traffics).Error; err != nil {
		return nil, err
	}
	return traffics, nil
}

func (r *GormInboundRepository) ListInvalidInboundTags(tx *gorm.DB, now int64) ([]string, error) {
	tags := make([]string, 0)
	if tx == nil {
		tx = r.db
	}
	if err := tx.Table("inbounds").
		Select("inbounds.tag").
		Where("((total > 0 and up + down >= total) or (expiry_time > 0 and expiry_time <= ?)) and enable = ?", now, true).
		Scan(&tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}

func (r *GormInboundRepository) DisableInvalidInbounds(tx *gorm.DB, now int64) (int64, error) {
	if tx == nil {
		tx = r.db
	}
	result := tx.Model(model.Inbound{}).
		Where("((total > 0 and up + down >= total) or (expiry_time > 0 and expiry_time <= ?)) and enable = ?", now, true).
		Update("enable", false)
	return result.RowsAffected, result.Error
}

func (r *GormInboundRepository) ListInvalidClientTrafficTargets(tx *gorm.DB, now int64) ([]ClientTrafficTarget, error) {
	targets := make([]ClientTrafficTarget, 0)
	if tx == nil {
		tx = r.db
	}
	if err := tx.Table("inbounds").
		Select("inbounds.tag, client_traffics.email").
		Joins("JOIN client_traffics ON inbounds.id = client_traffics.inbound_id").
		Where("((client_traffics.total > 0 AND client_traffics.up + client_traffics.down >= client_traffics.total) OR (client_traffics.expiry_time > 0 AND client_traffics.expiry_time <= ?)) AND client_traffics.enable = ?", now, true).
		Scan(&targets).Error; err != nil {
		return nil, err
	}
	return targets, nil
}

func (r *GormInboundRepository) DisableInvalidClientTraffics(tx *gorm.DB, now int64) (int64, error) {
	if tx == nil {
		tx = r.db
	}
	result := tx.Model(xray.ClientTraffic{}).
		Where("((total > 0 and up + down >= total) or (expiry_time > 0 and expiry_time <= ?)) and enable = ?", now, true).
		Update("enable", false)
	return result.RowsAffected, result.Error
}

func (r *GormInboundRepository) FindInboundBySettingsContains(query string) (*model.Inbound, error) {
	inbound := new(model.Inbound)
	if err := r.db.Model(model.Inbound{}).Where("settings LIKE ?", "%\""+query+"\"%").First(inbound).Error; err != nil {
		return nil, err
	}
	return inbound, nil
}

func (r *GormInboundRepository) SearchByRemark(query string) ([]*model.Inbound, error) {
	inbounds := make([]*model.Inbound, 0)
	if err := r.db.Model(model.Inbound{}).Preload("ClientStats").Where("remark like ?", "%"+query+"%").Find(&inbounds).Error; err != nil {
		return nil, err
	}
	return inbounds, nil
}

func (r *GormInboundRepository) ListTags() ([]string, error) {
	tags := make([]string, 0)
	if err := r.db.Model(model.Inbound{}).Select("tag").Find(&tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}

func (r *GormInboundRepository) DeleteOrphanedClientTraffics() error {
	return r.db.Exec(`
		DELETE FROM client_traffics
		WHERE email NOT IN (
			SELECT JSON_EXTRACT(client.value, '$.email')
			FROM inbounds,
				JSON_EACH(JSON_EXTRACT(inbounds.settings, '$.clients')) AS client
		)
	`).Error
}

func (r *GormInboundRepository) AddInboundTrafficByTag(tx *gorm.DB, tag string, upload int64, download int64) error {
	if tx == nil {
		tx = r.db
	}
	return tx.Model(&model.Inbound{}).Where("tag = ?", tag).
		Updates(map[string]any{
			"up":       gorm.Expr("up + ?", upload),
			"down":     gorm.Expr("down + ?", download),
			"all_time": gorm.Expr("COALESCE(all_time, 0) + ?", upload+download),
		}).Error
}

func (r *GormInboundRepository) UpdateClientTrafficUsageByEmail(email string, upload int64, download int64) error {
	return r.db.Model(xray.ClientTraffic{}).
		Where("email = ?", email).
		Updates(map[string]any{"up": upload, "down": download}).
		Error
}

func (r *GormInboundRepository) ResetClientTrafficByEmail(email string) error {
	return r.db.Model(xray.ClientTraffic{}).
		Where("email = ?", email).
		Updates(map[string]any{"enable": true, "up": 0, "down": 0}).
		Error
}

func (r *GormInboundRepository) SaveClientTraffic(traffic *xray.ClientTraffic) error {
	return r.db.Save(traffic).Error
}

func (r *GormInboundRepository) SaveClientTraffics(tx *gorm.DB, traffics []*xray.ClientTraffic) error {
	if len(traffics) == 0 {
		return nil
	}
	if tx == nil {
		tx = r.db
	}
	return tx.Save(traffics).Error
}

func (r *GormInboundRepository) ResetAllClientTraffics(id int) error {
	now := time.Now().Unix() * 1000
	return r.db.Transaction(func(tx *gorm.DB) error {
		whereText := "inbound_id "
		if id == -1 {
			whereText += "> ?"
		} else {
			whereText += "= ?"
		}

		result := tx.Model(xray.ClientTraffic{}).
			Where(whereText, id).
			Updates(map[string]any{"enable": true, "up": 0, "down": 0})
		if result.Error != nil {
			return result.Error
		}

		inboundWhereText := "id "
		if id == -1 {
			inboundWhereText += "> ?"
		} else {
			inboundWhereText += "= ?"
		}

		return tx.Model(model.Inbound{}).
			Where(inboundWhereText, id).
			Update("last_traffic_reset_time", now).
			Error
	})
}

func (r *GormInboundRepository) ResetAllTraffics() error {
	return r.db.Model(model.Inbound{}).
		Where("user_id > ?", 0).
		Updates(map[string]any{"up": 0, "down": 0}).
		Error
}

func (r *GormInboundRepository) ResetInboundTraffic(id int) error {
	return r.db.Model(model.Inbound{}).
		Where("id = ?", id).
		Updates(map[string]any{"up": 0, "down": 0}).
		Error
}

func (r *GormInboundRepository) CreateClientTraffic(tx *gorm.DB, clientTraffic *xray.ClientTraffic) error {
	return tx.Create(clientTraffic).Error
}

func (r *GormInboundRepository) UpdateClientTrafficByEmail(tx *gorm.DB, email string, client *model.Client) error {
	return tx.Model(xray.ClientTraffic{}).
		Where("email = ?", email).
		Updates(map[string]any{
			"enable":      client.Enable,
			"email":       client.Email,
			"total":       client.TotalGB,
			"expiry_time": client.ExpiryTime,
			"reset":       client.Reset,
		}).
		Error
}

func (r *GormInboundRepository) UpdateInboundClientIPs(tx *gorm.DB, oldEmail string, newEmail string) error {
	return tx.Model(model.InboundClientIps{}).Where("client_email = ?", oldEmail).Update("client_email", newEmail).Error
}

func (r *GormInboundRepository) FindInboundClientIps(clientEmail string) (*model.InboundClientIps, error) {
	ips := new(model.InboundClientIps)
	if err := r.db.Model(model.InboundClientIps{}).Where("client_email = ?", clientEmail).First(ips).Error; err != nil {
		return nil, err
	}
	return ips, nil
}

func (r *GormInboundRepository) ClearInboundClientIps(clientEmail string) error {
	return r.db.Model(model.InboundClientIps{}).Where("client_email = ?", clientEmail).Update("ips", "").Error
}

func (r *GormInboundRepository) DeleteClientTrafficByEmail(tx *gorm.DB, email string) error {
	return tx.Where("email = ?", email).Delete(xray.ClientTraffic{}).Error
}

func (r *GormInboundRepository) DeleteClientTrafficsByInboundID(tx *gorm.DB, inboundID int) error {
	return tx.Where("inbound_id = ?", inboundID).Delete(xray.ClientTraffic{}).Error
}

func (r *GormInboundRepository) DeleteInboundClientIPsByEmail(tx *gorm.DB, email string) error {
	return tx.Where("client_email = ?", email).Delete(model.InboundClientIps{}).Error
}

func (r *GormInboundRepository) DeleteByID(tx *gorm.DB, id int) error {
	return tx.Delete(model.Inbound{}, id).Error
}

func (r *GormInboundRepository) SaveInbound(tx *gorm.DB, inbound *model.Inbound) error {
	if tx == nil {
		tx = r.db
	}
	return tx.Save(inbound).Error
}

func (r *GormInboundRepository) SaveInbounds(tx *gorm.DB, inbounds []*model.Inbound) error {
	if len(inbounds) == 0 {
		return nil
	}
	if tx == nil {
		tx = r.db
	}
	return tx.Save(inbounds).Error
}

func (r *GormInboundRepository) Save(inbound *model.Inbound) error {
	return r.SaveInbound(r.db, inbound)
}

func inboundShadowsocksMethod(protocol string, settings string) string {
	if protocol != string(model.Shadowsocks) || settings == "" {
		return ""
	}
	var s struct {
		Method string `json:"method"`
	}
	if err := json.Unmarshal([]byte(settings), &s); err != nil {
		return ""
	}
	return s.Method
}

func inboundCanEnableTLSFlow(protocol string, streamSettings string) bool {
	if protocol != string(model.VLESS) || streamSettings == "" {
		return false
	}
	var stream struct {
		Network  string `json:"network"`
		Security string `json:"security"`
	}
	if err := json.Unmarshal([]byte(streamSettings), &stream); err != nil {
		return false
	}
	return stream.Network == "tcp" && (stream.Security == "tls" || stream.Security == "reality")
}
