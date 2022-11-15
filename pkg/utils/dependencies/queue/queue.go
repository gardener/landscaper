// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"fmt"
)

type Queue[T any] struct {
	elems []T
}

// New creates a new Queue.
func New[T any](elems ...T) Queue[T] {
	return Queue[T]{elems: elems}
}

// Append adds new elements to the queue.
func (sq *Queue[T]) Append(elems ...T) {
	sq.elems = append(sq.elems, elems...)
}

// Pop returns the first element and removes it from the queue.
// Returns an error if called on an empty queue.
func (sq *Queue[T]) Pop() (T, error) {
	res, err := sq.Peek()
	if err != nil {
		return res, err
	}
	sq.elems = sq.elems[1:]
	return res, nil
}

// Peek returns the first element of the queue without removing it.
// Returns an error if called on an empty queue.
func (sq *Queue[T]) Peek() (T, error) {
	if len(sq.elems) < 1 {
		var zero T
		return zero, fmt.Errorf("queue is empty")
	}
	return sq.elems[0], nil
}

// Len returns the length of the queue.
func (sq *Queue[_]) Len() int {
	return len(sq.elems)
}

// IsEmpty returns true if the queue is empty.
func (sq *Queue[_]) IsEmpty() bool {
	return sq.Len() == 0
}

// Copy returns a copy of the queue.
func (sq *Queue[T]) Copy() Queue[T] {
	return New(sq.elems...)
}
