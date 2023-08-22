// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package memstore

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCQueue(t *testing.T) {

	t.Run("new failing", func(t *testing.T) {
		q, err := NewCQueue[string](0)
		require.Nil(t, q)
		require.Error(t, err)
	})

	t.Run("new succeeding", func(t *testing.T) {
		q, err := NewCQueue[string](10)
		require.NotNil(t, q)
		require.NoError(t, err)
	})

	t.Run("get", func(t *testing.T) {
		q, err := NewCQueue[string](10)
		require.NotNil(t, q)
		require.NoError(t, err)

		var ptrs []*string
		for i := 0; i < 10; i++ {
			s := q.Get()
			require.NotNil(t, s)
			*s = fmt.Sprintf("test%d", i)
			ptrs = append(ptrs, s)
		}

		for i := 0; i < 10; i++ {
			s := q.Get()
			require.NotNil(t, s)
			require.Equal(t, fmt.Sprintf("test%d", i), *s)
			require.Equal(t, ptrs[i], s)
		}
	})

	t.Run("reset", func(t *testing.T) {
		q, err := NewCQueue[string](10)
		require.NotNil(t, q)
		require.NoError(t, err)

		var first *string
		for i := 0; i < 5; i++ {
			s := q.Get()
			require.NotNil(t, s)
			if i == 0 {
				first = s
			}
			*s = fmt.Sprintf("test%d", i)
		}

		q.Reset()
		s := q.Get()
		require.Equal(t, first, s)
		require.Equal(t, "test0", *s)
	})
}
