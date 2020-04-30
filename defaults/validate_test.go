package defaults

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	type serverConfiguration struct {
		URL           string `default:"http://localhost:8065" validate:"url"`
		Email         string `default:"sysadmin@sample.mattermost.com" validate:"email"`
		AdminPassword string `default:"Sys@dmin-sample1" validate:"text"`
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

}
