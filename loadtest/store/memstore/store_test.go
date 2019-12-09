package memstore

import (
	"testing"

	"github.com/mattermost/mattermost-server/model"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("NewMemStore", func(t *testing.T) {
		s := New()
		require.NotNil(t, s)
	})
}

func TestUser(t *testing.T) {
	s := New()

	t.Run("NilUser", func(t *testing.T) {
		u, err := s.User()
		require.NoError(t, err)
		require.Nil(t, u)
	})

	t.Run("SetUser", func(t *testing.T) {
		u := &model.User{}
		err := s.SetUser(u)
		require.NoError(t, err)
		uu, err := s.User()
		require.NoError(t, err)
		require.Equal(t, u, uu)
	})

	t.Run("SetPost", func(t *testing.T) {
		p := &model.Post{Id: model.NewId()}
		err := s.SetPost(p)
		require.NoError(t, err)
		uu, err := s.Post(p.Id)
		require.NoError(t, err)
		require.Equal(t, p, uu)
	})

	t.Run("SetTeam", func(t *testing.T) {
		tm := &model.Team{Id: model.NewId()}
		err := s.SetTeam(tm)
		require.NoError(t, err)
		tt, err := s.Team(tm.Id)
		require.NoError(t, err)
		require.Equal(t, tm, tt)
	})

	t.Run("SetTeams", func(t *testing.T) {
		tms := []*model.Team{
			{Id: model.NewId()},
			{Id: model.NewId()},
			{Id: model.NewId()},
		}
		err := s.SetTeams(tms)
		require.NoError(t, err)
		ttms, err := s.Teams()
		require.NoError(t, err)
		require.ElementsMatch(t, tms, ttms)
	})
}

func TestId(t *testing.T) {
	s := New()

	t.Run("EmptyId", func(t *testing.T) {
		id := s.Id()
		require.Empty(t, id)
	})

	t.Run("ExpectedId", func(t *testing.T) {
		expected := model.NewId()
		require.NoError(t, s.SetUser(&model.User{
			Id: expected,
		}))
		id := s.Id()
		require.Equal(t, expected, id)
	})
}
