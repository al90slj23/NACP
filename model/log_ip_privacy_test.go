package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func resetLogIpPrivacyTestTables(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.Exec("DELETE FROM logs").Error)
	require.NoError(t, DB.Exec("DELETE FROM users").Error)
	t.Cleanup(func() {
		_ = DB.Exec("DELETE FROM logs").Error
		_ = DB.Exec("DELETE FROM users").Error
	})
}

func TestUserLogIpVisibilityMasksStoredIpByUserSetting(t *testing.T) {
	resetLogIpPrivacyTestTables(t)

	user := User{
		Id:       61001,
		Username: "log_ip_hidden",
		Password: "password123",
		Status:   common.UserStatusEnabled,
		Role:     common.RoleCommonUser,
		Setting:  `{"record_ip_log":false}`,
	}
	require.NoError(t, DB.Create(&user).Error)
	require.NoError(t, LOG_DB.Create(&Log{
		UserId:    user.Id,
		Username:  user.Username,
		CreatedAt: 100,
		Type:      LogTypeConsume,
		Ip:        "203.0.113.10",
	}).Error)

	userLogs, _, err := GetUserLogs(user.Id, LogTypeUnknown, 0, 0, "", "", 0, 10, "", "")
	require.NoError(t, err)
	require.Len(t, userLogs, 1)
	require.Empty(t, userLogs[0].Ip)

	adminLogs, _, err := GetAllLogs(LogTypeUnknown, 0, 0, "", user.Username, "", 0, 10, 0, "", "")
	require.NoError(t, err)
	require.Len(t, adminLogs, 1)
	require.Equal(t, "203.0.113.10", adminLogs[0].Ip)
}

func TestUserLogIpVisibilityShowsStoredIpWhenEnabled(t *testing.T) {
	resetLogIpPrivacyTestTables(t)

	user := User{
		Id:       61002,
		Username: "log_ip_visible",
		Password: "password123",
		Status:   common.UserStatusEnabled,
		Role:     common.RoleCommonUser,
		Setting:  `{"record_ip_log":true}`,
	}
	require.NoError(t, DB.Create(&user).Error)
	require.NoError(t, LOG_DB.Create(&Log{
		UserId:    user.Id,
		Username:  user.Username,
		CreatedAt: 100,
		Type:      LogTypeConsume,
		Ip:        "203.0.113.20",
	}).Error)

	userLogs, _, err := GetUserLogs(user.Id, LogTypeUnknown, 0, 0, "", "", 0, 10, "", "")
	require.NoError(t, err)
	require.Len(t, userLogs, 1)
	require.Equal(t, "203.0.113.20", userLogs[0].Ip)
}
