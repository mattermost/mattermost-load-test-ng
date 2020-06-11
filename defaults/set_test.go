package defaults

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSet(t *testing.T) {
	t.Run("should fail on nil pointer", func(t *testing.T) {
		err := Set(nil)
		require.Error(t, err)
	})

	t.Run("should not fail on nil value", func(t *testing.T) {
		type st struct {
			String string `default:"text"`
		}
		var s st
		err := Set(&s)
		require.NoError(t, err)
		assert.Equal(t, "text", s.String)
	})

	t.Run("should fail on non-struct value", func(t *testing.T) {
		v := 1
		err := Set(&v)
		require.Error(t, err)
	})

	t.Run("should not fail on empty struct", func(t *testing.T) {
		err := Set(&struct{}{})
		require.NoError(t, err)
	})

	t.Run("should be able to set default values", func(t *testing.T) {
		cfg := struct {
			String        string  `default:"text"`
			StringNumeric string  `default:"123"`
			Integer       int     `default:"2"`
			Float64       float64 `default:"0.2"`
			Bool          bool    `default:"true"`
			AnotherStruct []struct {
				String string `default:"text"`
			} `default_size:"3"`
			YetAnotherStruct struct {
				String string `default:"text_other"`
			}
		}{}

		err := Set(&cfg)
		require.NoError(t, err)
		assert.Equal(t, "text", cfg.String)
		assert.Equal(t, "123", cfg.StringNumeric)
		assert.Equal(t, 2, cfg.Integer)
		assert.Equal(t, 0.2, cfg.Float64)
		assert.Equal(t, true, cfg.Bool)
		assert.Equal(t, "text", cfg.AnotherStruct[2].String)
		assert.Equal(t, "text_other", cfg.YetAnotherStruct.String)
	})

	t.Run("should fail on wrong default types", func(t *testing.T) {
		cfg := struct {
			Number int `default:"test"`
		}{}

		err := Set(&cfg)
		require.Error(t, err)
	})

	t.Run("should not fail for private fields", func(t *testing.T) {
		cfg := struct {
			integer int `default:"2"`
		}{}

		err := Set(&cfg)
		require.NoError(t, err)
		assert.Equal(t, 0, cfg.integer)
	})

	t.Run("should be able to set chan and map data types", func(t *testing.T) {
		cfg := struct {
			Map  map[int]string `default_size:"0"`
			Chan chan bool      `default_size:"3"`
		}{}

		err := Set(&cfg)
		require.NoError(t, err)
		assert.NotNil(t, cfg.Map)
		assert.NotNil(t, cfg.Chan)
		assert.Equal(t, 3, cap(cfg.Chan))
	})
}
