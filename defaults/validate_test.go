package defaults

import (
	"errors"
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
		err := Validate(cfg)
		require.Error(t, err)
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
}

type testConfig struct {
	Name string `default:"test" validate:"notempty"`
}

func (c *testConfig) IsValid() error {
	return errors.New("some error")
}
