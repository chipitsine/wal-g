package common_test

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/wal-g/wal-g/cmd/common"
	conf "github.com/wal-g/wal-g/internal/config"
)

func TestFlags_HiddenByDefault(t *testing.T) {
	rootCmd := &cobra.Command{Use: "wal-g"}
	common.Init(rootCmd, conf.PG)

	// Check persistent flags
	hiddenCount := 0
	visibleCount := 0
	rootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			hiddenCount++
		} else {
			visibleCount++
		}
	})

	assert.Greater(t, hiddenCount, 0, "expected config flags to be marked hidden")
	// --config should be visible
	assert.False(t, rootCmd.PersistentFlags().Lookup("config").Hidden)

	// Root usage should not contain hidden config flags
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	assert.NoError(t, rootCmd.Usage())
	usageOutput := buf.String()

	assert.Contains(t, usageOutput, "--config")
	assert.NotContains(t, usageOutput, "--walg-s3-prefix")
}

func TestFlagsCmd_UnhidesGlobalFlagsOnDemand(t *testing.T) {
	rootCmd := &cobra.Command{Use: "wal-g"}
	common.Init(rootCmd, conf.PG)

	// Before running FlagsCmd, config flags are hidden
	assert.True(t, rootCmd.PersistentFlags().Lookup("walg-s3-prefix").Hidden)

	// Request usage on FlagsCmd
	buf := new(bytes.Buffer)
	common.FlagsCmd.SetOut(buf)
	assert.NoError(t, common.FlagsCmd.Usage())

	flagsOutput := buf.String()
	assert.Contains(t, flagsOutput, "Global Flags:")
	assert.Contains(t, flagsOutput, "--walg-s3-prefix")

	// Verify that the flags are now unhidden
	assert.False(t, rootCmd.PersistentFlags().Lookup("walg-s3-prefix").Hidden)
}

func TestFlags_CanBeParsedAndBoundToViper(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	rootCmd := &cobra.Command{
		Use: "wal-g",
		Run: func(cmd *cobra.Command, args []string) {},
	}
	common.Init(rootCmd, conf.PG)

	rootCmd.SetArgs([]string{"--walg-s3-prefix=s3://my-test-bucket/prefix"})
	assert.NoError(t, rootCmd.Execute())

	assert.Equal(t, "s3://my-test-bucket/prefix", viper.GetString("WALG_S3_PREFIX"))
}
