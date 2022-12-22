package defaults

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	type serverConfiguration struct {
		URL           string `default:"http://localhost:8065" validate:"url"`
		Email         string `default:"sysadmin@sample.mattermost.com" validate:"email"`
		AdminPassword string `default:"Sys@dmin-sample1" validate:"notempty"`
		NumTeams      int    `default:"2" validate:"range:(0,]"`
		InitialUsers  int    `default:"0" validate:"range:[0,$MaxUsers]"`
		MaxUsers      int    `default:"1000" validate:"range:(0,]"`
		LogLevel      string `default:"ERROR" validate:"oneof:{TRACE, INFO, WARN, ERROR}"`
		S3URI         string `default:"" validate:"s3uri"`
	}

	t.Run("happy path", func(t *testing.T) {
		var cfg serverConfiguration
		Set(&cfg)

		err := Validate(cfg)
		require.NoError(t, err)
	})

	t.Run("happy path with ptr", func(t *testing.T) {
		var cfg serverConfiguration
		Set(&cfg)

		err := Validate(&cfg)
		require.NoError(t, err)
	})

	t.Run("out of given range", func(t *testing.T) {
		var cfg serverConfiguration
		Set(&cfg)

		cfg.InitialUsers = -1
		err := Validate(cfg)
		require.Error(t, err)
	})

	t.Run("invalid url", func(t *testing.T) {
		var cfg serverConfiguration
		Set(&cfg)

		cfg.URL = "localhost"
		err := Validate(cfg)
		require.Error(t, err)
	})

	t.Run("invalid email", func(t *testing.T) {
		var cfg serverConfiguration
		Set(&cfg)

		cfg.Email = "some_text"
		err := Validate(cfg)
		require.Error(t, err)
	})

	t.Run("not one of", func(t *testing.T) {
		var cfg serverConfiguration
		Set(&cfg)

		cfg.LogLevel = "DEBUG"
		valids := []string{"TRACE", "INFO", "WARN", "ERROR"}
		err := Validate(cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), fmt.Sprintf("% q", valids))
	})

	t.Run("invalid field reference", func(t *testing.T) {
		type invalidField struct {
			InitialUsers int `default:"0" validate:"range:[0,$MaxActiveUsers]"`
		}

		var cfg invalidField
		Set(&cfg)

		err := Validate(cfg)
		require.Error(t, err)
	})

	t.Run("call IsValid method", func(t *testing.T) {
		var cfg testConfig

		err := Set(&cfg)
		require.NoError(t, err)

		err = Validate(&cfg)
		require.Error(t, err)
	})

	t.Run("invalid struct tags", func(t *testing.T) {
		type missingColonConfig struct {
			InitialUsers int `default:"0" validate:"oneof:{test,text}"`
		}

		var cfg1 missingColonConfig
		Set(&cfg1)

		err := Validate(cfg1)
		require.Error(t, err, "should fail on missing colon")

		type invalidField struct {
			Name         string `default:"test" validate:"notempty"`
			InitialUsers int    `default:"0" validate:"range:[0,$Name]"`
		}

		var cfg2 invalidField
		Set(&cfg2)

		err = Validate(cfg2)
		require.Error(t, err, "should fail on wrong type")

		type invalidRange struct {
			InitialUsers int `default:"-1" validate:"range:[,(0,123),]"`
		}

		var cfg3 invalidRange
		Set(&cfg3)

		err = Validate(cfg3)
		require.Error(t, err, "should fail on wrong range declaration")
	})

	t.Run("notempty on slices", func(t *testing.T) {
		type testStruct struct {
			Slice []struct{} `validate:"notempty"`
		}

		t1 := testStruct{make([]struct{}, 0)}
		t2 := testStruct{make([]struct{}, 1)}

		err := Validate(t1)
		require.Error(t, err)
		err = Validate(t2)
		require.NoError(t, err)
	})

	t.Run("valid S3 URI", func(t *testing.T) {
		var cfg serverConfiguration
		Set(&cfg)

		cfg.S3URI = "s3://test.s3bucket"

		err := Validate(&cfg)
		require.NoError(t, err)
	})

	t.Run("invalid S3 URI", func(t *testing.T) {
		var cfg serverConfiguration
		Set(&cfg)

		cfg.S3URI = "not an url"

		err := Validate(&cfg)
		require.Error(t, err)
	})

	t.Run("invalid S3 URI scheme", func(t *testing.T) {
		var cfg serverConfiguration
		Set(&cfg)

		cfg.S3URI = "https://validurl.com/but/wrong/scheme"

		err := Validate(&cfg)
		require.Error(t, err)
	})
}

type testConfig struct {
	Name string `default:"test" validate:"notempty"`
}

func (c *testConfig) IsValid() error {
	return errors.New("some error")
}

func TestValidateComparisonConfig(t *testing.T) {
	type LoadTestType string
	type DatabaseEngine string
	type LoadTestConfig struct {
		Type                  LoadTestType   `validate:"oneof:{bounded,unbounded}"`
		DBEngine              DatabaseEngine `validate:"oneof:{mysql,postgresql}"`
		DBDumpURL             string
		PermalinkIPsToReplace []string
		S3BucketDumpURI       string `default:"" validate:"s3uri"`
		NumUsers              int    `default:"0" validate:"range:[0,]"`
		Duration              string
	}

	config := LoadTestConfig{
		Type:      "bounded",
		DBEngine:  "postgresql",
		DBDumpURL: "file:///home/ubuntu/dump.sql",
		PermalinkIPsToReplace: []string{
			"44.201.217.130",
			"52.87.227.97",
			"52.91.86.20",
			"54.174.96.187",
		},
		S3BucketDumpURI: "s3://test.bucket/subdir",
		NumUsers:        100,
		Duration:        "1h",
	}

	t.Run("valid config", func(t *testing.T) {
		err := Validate(config)
		require.NoError(t, err)
	})
}
