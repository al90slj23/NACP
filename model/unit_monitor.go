package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	UnitMonitorStatusEnabled  = 1
	UnitMonitorStatusDisabled = 2
)

type UnitAccountMonitor struct {
	Id               int     `json:"id"`
	UnitID           int     `json:"unit_id" gorm:"index;not null;uniqueIndex:idx_unit_account_monitor"`
	UnitAccountID    int     `json:"unit_account_id" gorm:"index;not null;uniqueIndex:idx_unit_account_monitor"`
	Name             string  `json:"name" gorm:"type:varchar(180);default:''"`
	Status           int     `json:"status" gorm:"default:1;index"`
	CurrentBalance   float64 `json:"current_balance" gorm:"default:0"`
	UsedAmount       float64 `json:"used_amount" gorm:"default:0"`
	BalanceUnit      string  `json:"balance_unit" gorm:"type:varchar(32);default:'USD'"`
	PlatformStatus   string  `json:"platform_status" gorm:"type:varchar(64);default:'pending';index"`
	UpstreamUserID   string  `json:"upstream_user_id" gorm:"type:varchar(180);default:''"`
	UpstreamUsername string  `json:"upstream_username" gorm:"type:varchar(180);default:''"`
	UpstreamGroup    string  `json:"upstream_group" gorm:"type:varchar(180);default:''"`
	TokenCount       int     `json:"token_count" gorm:"default:0"`
	ModelCount       int     `json:"model_count" gorm:"default:0"`
	GroupCount       int     `json:"group_count" gorm:"default:0"`
	LastCheckedTime  int64   `json:"last_checked_time" gorm:"bigint;default:0;index"`
	ErrorMessage     string  `json:"error_message" gorm:"type:text"`
	RawJSON          string  `json:"raw_json" gorm:"type:text"`
	CreatedTime      int64   `json:"created_time" gorm:"bigint"`
	UpdatedTime      int64   `json:"updated_time" gorm:"bigint"`

	UnitName   string `json:"unit_name" gorm:"-"`
	UnitType   string `json:"unit_type" gorm:"-"`
	UnitAPIURL string `json:"unit_api_url" gorm:"-"`
	Account    string `json:"account" gorm:"-"`
	AccountID  string `json:"account_id" gorm:"-"`
}

type UnitAccountMonitorSnapshot struct {
	Id               int     `json:"id"`
	MonitorID        int     `json:"monitor_id" gorm:"index;not null"`
	UnitID           int     `json:"unit_id" gorm:"index;not null"`
	UnitAccountID    int     `json:"unit_account_id" gorm:"index;not null"`
	CurrentBalance   float64 `json:"current_balance" gorm:"default:0"`
	UsedAmount       float64 `json:"used_amount" gorm:"default:0"`
	BalanceUnit      string  `json:"balance_unit" gorm:"type:varchar(32);default:'USD'"`
	PlatformStatus   string  `json:"platform_status" gorm:"type:varchar(64);default:'ok';index"`
	LastCheckedTime  int64   `json:"last_checked_time" gorm:"bigint;default:0;index"`
	CreatedTime      int64   `json:"created_time" gorm:"bigint;index"`
	UpstreamUserID   string  `json:"upstream_user_id" gorm:"type:varchar(180);default:''"`
	UpstreamUsername string  `json:"upstream_username" gorm:"type:varchar(180);default:''"`
}

func (m *UnitAccountMonitor) Normalize() {
	m.Name = strings.TrimSpace(m.Name)
	if m.Status == 0 {
		m.Status = UnitMonitorStatusEnabled
	}
	if strings.TrimSpace(m.BalanceUnit) == "" {
		m.BalanceUnit = "USD"
	}
}

func (m *UnitAccountMonitor) Insert() error {
	m.Normalize()
	now := common.GetTimestamp()
	m.CreatedTime = now
	m.UpdatedTime = now
	return DB.Create(m).Error
}

func (m *UnitAccountMonitor) UpdateBasic() error {
	m.Normalize()
	m.UpdatedTime = common.GetTimestamp()
	return DB.Model(&UnitAccountMonitor{}).Where("id = ?", m.Id).
		Select("name", "status", "updated_time").
		Updates(m).Error
}

func (m *UnitAccountMonitor) SaveCheckResult() error {
	m.UpdatedTime = common.GetTimestamp()
	return DB.Model(&UnitAccountMonitor{}).Where("id = ?", m.Id).
		Select(
			"current_balance",
			"used_amount",
			"balance_unit",
			"platform_status",
			"upstream_user_id",
			"upstream_username",
			"upstream_group",
			"token_count",
			"model_count",
			"group_count",
			"last_checked_time",
			"error_message",
			"raw_json",
			"updated_time",
		).
		Updates(m).Error
}

func (m *UnitAccountMonitor) RecordSnapshot() error {
	if m == nil || m.Id == 0 {
		return nil
	}
	snapshot := UnitAccountMonitorSnapshot{
		MonitorID:        m.Id,
		UnitID:           m.UnitID,
		UnitAccountID:    m.UnitAccountID,
		CurrentBalance:   m.CurrentBalance,
		UsedAmount:       m.UsedAmount,
		BalanceUnit:      m.BalanceUnit,
		PlatformStatus:   m.PlatformStatus,
		LastCheckedTime:  m.LastCheckedTime,
		CreatedTime:      common.GetTimestamp(),
		UpstreamUserID:   m.UpstreamUserID,
		UpstreamUsername: m.UpstreamUsername,
	}
	return DB.Create(&snapshot).Error
}

func (m *UnitAccountMonitor) Delete() error {
	return DB.Delete(m).Error
}

func GetUnitAccountMonitorById(id int) (*UnitAccountMonitor, error) {
	monitor := UnitAccountMonitor{Id: id}
	err := DB.First(&monitor, "id = ?", id).Error
	if err != nil {
		return &monitor, err
	}
	fillUnitMonitorDisplayFields(&monitor)
	return &monitor, nil
}

func GetUnitAccountMonitorByAccount(unitID int, accountID int) (*UnitAccountMonitor, error) {
	monitor := UnitAccountMonitor{}
	err := DB.Where("unit_id = ? AND unit_account_id = ?", unitID, accountID).First(&monitor).Error
	if err != nil {
		return &monitor, err
	}
	fillUnitMonitorDisplayFields(&monitor)
	return &monitor, nil
}

func SearchUnitAccountMonitors(keyword string, status int, unitID int, unitAccountID int, offset int, limit int) ([]*UnitAccountMonitor, int64, error) {
	var monitors []*UnitAccountMonitor
	db := DB.Model(&UnitAccountMonitor{})
	if status > 0 {
		db = db.Where("status = ?", status)
	}
	if unitID > 0 {
		db = db.Where("unit_id = ?", unitID)
	}
	if unitAccountID > 0 {
		db = db.Where("unit_account_id = ?", unitAccountID)
	}
	if keyword != "" {
		like := "%" + strings.TrimSpace(keyword) + "%"
		db = db.Where("name LIKE ? OR upstream_username LIKE ? OR upstream_user_id LIKE ?", like, like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("id DESC").Offset(offset).Limit(limit).Find(&monitors).Error; err != nil {
		return nil, 0, err
	}
	for _, monitor := range monitors {
		fillUnitMonitorDisplayFields(monitor)
	}
	return monitors, total, nil
}

func ListEnabledUnitAccountMonitors() ([]*UnitAccountMonitor, error) {
	var monitors []*UnitAccountMonitor
	err := DB.Where("status = ?", UnitMonitorStatusEnabled).Order("id ASC").Find(&monitors).Error
	return monitors, err
}

func fillUnitMonitorDisplayFields(monitor *UnitAccountMonitor) {
	if monitor == nil {
		return
	}
	unit := Unit{}
	if err := DB.First(&unit, "id = ?", monitor.UnitID).Error; err == nil {
		unit.FillEffectiveAPIURL()
		monitor.UnitName = unit.Name
		monitor.UnitType = unit.Type
		monitor.UnitAPIURL = unit.EffectiveAPIURL
	}
	account := UnitAccount{}
	if err := DB.First(&account, "id = ?", monitor.UnitAccountID).Error; err == nil {
		monitor.Account = account.Account
		monitor.AccountID = account.AccountID
	}
}

func IsUnitAccountMonitorNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
