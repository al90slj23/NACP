package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	UnitStatusEnabled  = 1
	UnitStatusDisabled = 2

	UnitTypeNewAPI      = "newapi"
	UnitTypeSub2API     = "sub2api"
	UnitTypeOneAPI      = "oneapi"
	UnitTypeOneAPIFork  = "oneapifork"
	UnitTypeShellAPI    = "shellapi"
	UnitTypeRixAPI      = "rixapi"
	UnitTypeVeloera     = "veloera"
	UnitTypeOneHub      = "onehub"
	UnitTypeDoneHub     = "donehub"
	UnitTypeAnyRouter   = "anyrouter"
	UnitTypeOpenAI      = "openai"
	UnitTypeClaude      = "claude"
	UnitTypeGemini      = "gemini"
	UnitTypeGeminiCLI   = "geminicli"
	UnitTypeAntigravity = "antigravity"
	UnitTypeCLIProxyAPI = "cliproxyapi"
	UnitTypeCodex       = "codex"
	UnitTypeOther       = "other"
)

type Unit struct {
	Id              int           `json:"id"`
	Name            string        `json:"name" gorm:"type:varchar(128);not null;index"`
	Remark          string        `json:"remark" gorm:"type:text"`
	WebsiteURL      string        `json:"website_url" gorm:"type:varchar(512);default:''"`
	APIURL          string        `json:"api_url" gorm:"type:varchar(512);default:''"`
	Type            string        `json:"type" gorm:"type:varchar(64);default:'newapi';index"`
	Status          int           `json:"status" gorm:"default:1;index"`
	CreatedTime     int64         `json:"created_time" gorm:"bigint"`
	UpdatedTime     int64         `json:"updated_time" gorm:"bigint"`
	EffectiveAPIURL string        `json:"effective_api_url" gorm:"-"`
	AccountCount    int64         `json:"account_count" gorm:"-"`
	Accounts        []UnitAccount `json:"accounts,omitempty" gorm:"-"`
}

type UnitAccount struct {
	Id          int    `json:"id"`
	UnitID      int    `json:"unit_id" gorm:"index;not null"`
	Account     string `json:"account" gorm:"type:varchar(180);default:''"`
	Password    string `json:"password" gorm:"type:varchar(255);default:''"`
	AccessToken string `json:"access_token" gorm:"type:text"`
	AccountID   string `json:"account_id" gorm:"type:varchar(180);default:''"`
	CreatedTime int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime int64  `json:"updated_time" gorm:"bigint"`
}

func (u *Unit) Normalize() {
	u.Name = strings.TrimSpace(u.Name)
	u.WebsiteURL = strings.TrimSpace(u.WebsiteURL)
	u.APIURL = strings.TrimSpace(u.APIURL)
	u.Type = strings.TrimSpace(strings.ToLower(u.Type))
	if u.Type == "" {
		u.Type = UnitTypeNewAPI
	}
	if u.Status == 0 {
		u.Status = UnitStatusEnabled
	}
	u.FillEffectiveAPIURL()
}

func (u *Unit) FillEffectiveAPIURL() {
	if strings.TrimSpace(u.APIURL) != "" {
		u.EffectiveAPIURL = strings.TrimSpace(u.APIURL)
		return
	}
	u.EffectiveAPIURL = strings.TrimSpace(u.WebsiteURL)
}

func (u *Unit) Insert() error {
	u.Normalize()
	now := common.GetTimestamp()
	u.CreatedTime = now
	u.UpdatedTime = now
	return DB.Create(u).Error
}

func (u *Unit) Update() error {
	u.Normalize()
	u.UpdatedTime = common.GetTimestamp()
	return DB.Model(&Unit{}).Where("id = ?", u.Id).
		Select("name", "remark", "website_url", "api_url", "type", "status", "updated_time").
		Updates(u).Error
}

func (u *Unit) Delete() error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("unit_id = ?", u.Id).Delete(&UnitAccount{}).Error; err != nil {
			return err
		}
		return tx.Delete(u).Error
	})
}

func GetUnitById(id int) (*Unit, error) {
	unit := Unit{Id: id}
	err := DB.First(&unit, "id = ?", id).Error
	if err != nil {
		return &unit, err
	}
	unit.FillEffectiveAPIURL()
	accounts, err := GetUnitAccounts(id)
	if err != nil {
		return &unit, err
	}
	unit.Accounts = accounts
	unit.AccountCount = int64(len(accounts))
	return &unit, err
}

func SearchUnits(keyword string, unitType string, status int, offset int, limit int) ([]*Unit, int64, error) {
	var units []*Unit
	db := DB.Model(&Unit{})
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name LIKE ? OR remark LIKE ? OR website_url LIKE ? OR api_url LIKE ?", like, like, like, like)
	}
	if unitType != "" {
		db = db.Where("type = ?", strings.TrimSpace(strings.ToLower(unitType)))
	}
	if status > 0 {
		db = db.Where("status = ?", status)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("id DESC").Offset(offset).Limit(limit).Find(&units).Error; err != nil {
		return nil, 0, err
	}
	for _, unit := range units {
		unit.FillEffectiveAPIURL()
		unit.AccountCount = CountUnitAccounts(unit.Id)
	}
	return units, total, nil
}

func (a *UnitAccount) Normalize(unitID int) {
	a.UnitID = unitID
	a.Account = strings.TrimSpace(a.Account)
	a.Password = strings.TrimSpace(a.Password)
	a.AccessToken = strings.TrimSpace(a.AccessToken)
	a.AccountID = strings.TrimSpace(a.AccountID)
}

func (a *UnitAccount) IsEmpty() bool {
	return a.Account == "" && a.Password == "" && a.AccessToken == "" && a.AccountID == ""
}

func GetUnitAccounts(unitID int) ([]UnitAccount, error) {
	accounts := make([]UnitAccount, 0)
	err := DB.Where("unit_id = ?", unitID).Order("id ASC").Find(&accounts).Error
	return accounts, err
}

func GetUnitAccountById(id int) (*UnitAccount, error) {
	account := UnitAccount{Id: id}
	err := DB.First(&account, "id = ?", id).Error
	return &account, err
}

func CountUnitAccounts(unitID int) int64 {
	var count int64
	_ = DB.Model(&UnitAccount{}).Where("unit_id = ?", unitID).Count(&count).Error
	return count
}

func SyncUnitAccounts(unitID int, accounts []UnitAccount) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var existing []UnitAccount
		if err := tx.Where("unit_id = ?", unitID).Find(&existing).Error; err != nil {
			return err
		}

		keepIds := make([]int, 0, len(accounts))
		now := common.GetTimestamp()
		for i := range accounts {
			accounts[i].Normalize(unitID)
			if accounts[i].IsEmpty() {
				continue
			}
			if accounts[i].Id > 0 {
				keepIds = append(keepIds, accounts[i].Id)
				if err := tx.Model(&UnitAccount{}).
					Where("id = ? AND unit_id = ?", accounts[i].Id, unitID).
					Updates(map[string]interface{}{
						"account":      accounts[i].Account,
						"password":     accounts[i].Password,
						"access_token": accounts[i].AccessToken,
						"account_id":   accounts[i].AccountID,
						"updated_time": now,
					}).Error; err != nil {
					return err
				}
				continue
			}
			accounts[i].CreatedTime = now
			accounts[i].UpdatedTime = now
			if err := tx.Create(&accounts[i]).Error; err != nil {
				return err
			}
			keepIds = append(keepIds, accounts[i].Id)
		}

		for _, item := range existing {
			found := false
			for _, id := range keepIds {
				if item.Id == id {
					found = true
					break
				}
			}
			if !found {
				if err := tx.Delete(&UnitAccount{}, item.Id).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
