// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package memstore

import (
	"errors"
)

// CQueue is a basic implementation of a circular queue of fixed size.
type CQueue[T any] struct {
	data []*T // The backing data slice.
	size int  // The maximum size (capacity) of the backing data slice.
	next int  // The index of the next element to return on a call to Get().
}

// NewCQueue creates and returns a pointer to a queue of the given size.
func NewCQueue[T any](size int) (*CQueue[T], error) {
	if size <= 0 {
		return nil, errors.New("size should be > 0")
	}
	q := &CQueue[T]{
		data: make([]*T, 0, size),
		size: size,
		next: 0,
	}
	return q, nil
}

// Get returns a pointer to the next element in the queue.
// In case this is nil, data for it will be allocated before returning.
func (q *CQueue[T]) Get() *T {
	if q.next == len(q.data) {
		q.data = append(q.data, nil)
	}
	if q.data[q.next] == nil {
		q.data[q.next] = new(T)
	}
	el := q.data[q.next]
	q.next++
	if q.next == q.size {
		q.next = 0
	}
	return el
}

// Reset re-initializes the queue to be reused.
func (q *CQueue[T]) Reset() {
	q.next = 0
}
