package simplecontroller

import (
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/stretchr/testify/require"
)

func TestSetRate(t *testing.T) {
	c := SimpleController{}

	c.Init(&userentity.UserEntity{})
	require.Equal(t, 1.0, c.rate)

	err := c.SetRate(-1.0)
	require.NotNil(t, err)
	require.Equal(t, 1.0, c.rate)

	err = c.SetRate(0.0)
	require.Nil(t, err)
	require.Equal(t, 0.0, c.rate)

	err = c.SetRate(1.5)
	require.Nil(t, err)
	require.Equal(t, 1.5, c.rate)
}
