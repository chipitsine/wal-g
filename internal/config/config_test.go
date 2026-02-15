package config_test

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/wal-g/tracelog"
	"github.com/wal-g/wal-g/internal"
	"github.com/wal-g/wal-g/internal/config"
)

func TestGetMaxConcurrency_InvalidKey(t *testing.T) {
	_, err := config.GetMaxConcurrency("INVALID_KEY")

	assert.Error(t, err)
}

func TestGetMaxConcurrency_ValidKey(t *testing.T) {
	viper.Set(config.UploadConcurrencySetting, "100")
	actual, err := config.GetMaxConcurrency(config.UploadConcurrencySetting)

	assert.NoError(t, err)
	assert.Equal(t, 100, actual)
	resetToDefaults()
}

func TestGetMaxConcurrency_ValidKeyAndNegativeValue(t *testing.T) {
	viper.Set(config.UploadConcurrencySetting, "-5")
	_, err := config.GetMaxConcurrency(config.UploadConcurrencySetting)

	assert.Error(t, err)
	resetToDefaults()
}

func TestGetMaxConcurrency_ValidKeyAndInvalidValue(t *testing.T) {
	viper.Set(config.UploadConcurrencySetting, "invalid")
	_, err := config.GetMaxConcurrency(config.UploadConcurrencySetting)

	assert.Error(t, err)
	resetToDefaults()
}

func TestConfigureLogging_WhenLogLevelSettingIsNotSet(t *testing.T) {
	assert.NoError(t, config.ConfigureLogging())
}

func TestConfigureLogging_WhenLogLevelSettingIsSet(t *testing.T) {
	viper.Set(config.LogLevelSetting, "someOtherLevel")
	err := config.ConfigureLogging()

	assert.Error(t, err)
	assert.Error(t, tracelog.Setup(os.Stderr, viper.GetString(config.LogLevelSetting)))
	resetToDefaults()
}

func TestConfigureLogging_WhenLogDestinationSettingIsSet(t *testing.T) {
	viper.Set(config.LogLevelSetting, "/some/nonexistent/file")
	err := config.ConfigureLogging()

	assert.Error(t, err)
	resetToDefaults()
}

func TestSQLServerFailoverStorageSettings(t *testing.T) {
	viper.Reset()
	internal.ConfigureSettings(config.SQLSERVER)
	config.InitConfig()

	// Verify that failover storage settings are in the AllowedSettings map
	assert.True(t, config.AllowedSettings[config.FailoverStoragesCheckTimeout],
		"WALG_FAILOVER_STORAGES_CHECK_TIMEOUT should be allowed for SQL Server")
	assert.True(t, config.AllowedSettings[config.FailoverStorageCacheLifetime],
		"WALG_FAILOVER_STORAGES_CACHE_LIFETIME should be allowed for SQL Server")
	assert.True(t, config.AllowedSettings[config.FailoverStorages],
		"WALG_FAILOVER_STORAGES should be allowed for SQL Server")
	assert.True(t, config.AllowedSettings[config.FailoverStoragesCheck],
		"WALG_FAILOVER_STORAGES_CHECK should be allowed for SQL Server")

	resetToDefaults()
}

func resetToDefaults() {
	viper.Reset()
	internal.ConfigureSettings(config.PG)
	config.InitConfig()
	config.Configure()
}
