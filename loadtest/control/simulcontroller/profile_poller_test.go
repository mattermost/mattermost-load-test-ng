// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simulcontroller

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestProfilePollerBatchesAndDeduplicates(t *testing.T) {
	type call struct{ users, statuses []string }
	var mut sync.Mutex
	var calls []call
	p := newProfilePoller(time.Hour, func(u, s []string) {
		mut.Lock()
		defer mut.Unlock()
		calls = append(calls, call{u, s})
	})

	// 250 queues of overlapping ids collapse into the unique set.
	for i := 0; i < 250; i++ {
		p.queue([]string{fmt.Sprintf("u%d", i%150)}, []string{fmt.Sprintf("u%d", i%150)})
	}
	require.Empty(t, calls, "nothing is fetched before the interval")

	// 150 users and statuses pending: users are capped at 100 per request.
	p.flush()
	require.Len(t, calls, 1)
	require.Len(t, calls[0].users, maxUserIdsPerProfiles)
	require.Len(t, calls[0].statuses, 150)

	// The rest goes in the next interval, and then nothing is left.
	p.flush()
	require.Len(t, calls, 2)
	require.Len(t, calls[1].users, 50)
	require.Empty(t, calls[1].statuses)
	p.flush()
	require.Len(t, calls, 2)
}

func TestProfilePollerFiresAfterInterval(t *testing.T) {
	done := make(chan struct{})
	p := newProfilePoller(20*time.Millisecond, func(u, s []string) { close(done) })
	p.queue([]string{"a"}, nil)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("poller did not fire")
	}
}

func TestProfilePollerReset(t *testing.T) {
	fired := false
	p := newProfilePoller(20*time.Millisecond, func(u, s []string) { fired = true })
	p.queue([]string{"a"}, []string{"a"})
	p.reset()
	time.Sleep(100 * time.Millisecond)
	require.False(t, fired)

	// It is usable again after a reset.
	done := make(chan struct{})
	p.fetch = func(u, s []string) { close(done) }
	p.queue([]string{"b"}, nil)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("poller did not fire after reset")
	}
}
