// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package memstore

import (
	"errors"
)

// CQueue is a basic implementation of a circular queue of fixed size.
type CQueue struct {
	data []interface{}
	new  func() interface{}
	size int
	next int
}

// NewCQueue creates and returns a pointer to a queue of the given size.
// The passed new parameter is a function used to allocate an element if data
// for it is not yet present in the queue.
func NewCQueue(size int, new func() interface{}) (*CQueue, error) {
	if size <= 0 {
		return nil, errors.New("size should be > 0")
	}
	if new == nil {
		return nil, errors.New("new should not be nil")
	}
	q := &CQueue{
		data: make([]interface{}, 0, size),
		new:  new,
		size: size,
		next: 0,
	}
	return q, nil
}

// Get returns a pointer to the next element in the queue.
// In case this is nil, data for it will be allocated before returning.
func (q *CQueue) Get() interface{} {
	if q.next == len(q.data) {
		q.data = append(q.data, nil)
	}
	if q.data[q.next] == nil {
		q.data[q.next] = q.new()
	}
	el := q.data[q.next]
	q.next++
	if q.next == q.size {
		q.next = 0
	}
	return el
}

// Reset re-initializes the queue to be reused.
func (q *CQueue) Reset() {
	q.next = 0
}
