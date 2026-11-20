// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simulcontroller

import (
	"sync"
	"time"
)

const (
	// The webapp does not fetch unknown users and statuses one by one: it
	// queues their ids and a background loader requests them in batches every
	// UsersStatusAndProfileFetchingPollIntervalMilliseconds (3 seconds by
	// default), at most 100 users and 200 statuses per request.
	// See addUserIdsForProfileFetchingPoll and addUserIdsForStatusFetchingPoll
	// in webapp/channels/src/packages/mattermost-redux/src/actions/status_profile_polling.ts.
	profilePollInterval   = 3 * time.Second
	maxUserIdsPerProfiles = 100
	maxUserIdsPerStatuses = 200
)

// profilePoller models the webapp's polling data loaders for user profiles and
// statuses. Ids are queued (and deduplicated) and a single fetch per interval
// requests everything that is pending.
type profilePoller struct {
	interval time.Duration
	fetch    func(userIds, statusIds []string)

	mut       sync.Mutex
	users     map[string]struct{}
	statuses  map[string]struct{}
	scheduled bool
	timer     *time.Timer
}

func newProfilePoller(interval time.Duration, fetch func(userIds, statusIds []string)) *profilePoller {
	return &profilePoller{
		interval: interval,
		fetch:    fetch,
		users:    map[string]struct{}{},
		statuses: map[string]struct{}{},
	}
}

// queue adds ids to the pending sets and starts the timer if needed.
func (p *profilePoller) queue(userIds, statusIds []string) {
	p.mut.Lock()
	defer p.mut.Unlock()

	for _, id := range userIds {
		p.users[id] = struct{}{}
	}
	for _, id := range statusIds {
		p.statuses[id] = struct{}{}
	}
	p.scheduleLocked()
}

func (p *profilePoller) scheduleLocked() {
	if p.scheduled || (len(p.users) == 0 && len(p.statuses) == 0) {
		return
	}
	p.scheduled = true
	p.timer = time.AfterFunc(p.interval, p.flush)
}

func take(set map[string]struct{}, max int) []string {
	n := min(len(set), max)
	ids := make([]string, 0, n)
	for id := range set {
		ids = append(ids, id)
		delete(set, id)
		if len(ids) >= max {
			break
		}
	}
	return ids
}

// flush requests one batch of pending users and statuses. When more ids are
// pending than fit in a batch, the next one waits for the next interval.
func (p *profilePoller) flush() {
	p.mut.Lock()
	p.scheduled = false
	users := take(p.users, maxUserIdsPerProfiles)
	statuses := take(p.statuses, maxUserIdsPerStatuses)
	p.scheduleLocked()
	p.mut.Unlock()

	if len(users) > 0 || len(statuses) > 0 {
		p.fetch(users, statuses)
	}
}

// reset drops everything pending and cancels the timer. The poller can be
// used again afterwards.
func (p *profilePoller) reset() {
	p.mut.Lock()
	defer p.mut.Unlock()

	if p.timer != nil {
		p.timer.Stop()
	}
	p.scheduled = false
	clear(p.users)
	clear(p.statuses)
}
