package util

import (
	"iter"
	"maps"
	"slices"
)

// Set is a true set of items implemented using a map
type Set[T comparable] map[T]struct{}

// NewSet creates a new empty set
func NewSet[T comparable]() Set[T] {
	return make(Set[T])
}

// Has checks if a Set contains an element
func (s Set[T]) Has(check T) bool {
	_, present := s[check]
	return present
}

// Add adds an element to a Set
func (s Set[T]) Add(value T) {
	s[value] = struct{}{}
}

// Remove drops an element from a Set
func (s Set[T]) Remove(value T) {
	delete(s, value)
}

// All returns an iterator for the set
func (s Set[T]) All() iter.Seq[T] {
	return maps.Keys(s)
}

// ToSlice converts a set into a slice
func (s Set[T]) ToSlice() []T {
	return slices.Collect(s.All())
}

// Len returns the number of elements in the set
func (s Set[T]) Len() int {
	return len(s)
}

// Clear removes all elements from the set
func (s Set[T]) Clear() {
	clear(s)
}

// SetFromSlice takes a slice and returns a Set
func SetFromSlice[T comparable](source []T) Set[T] {
	s := NewSet[T]()
	for _, val := range source {
		s.Add(val)
	}
	return s
}

// SetEqual compares two Sets
func SetEqual[T comparable](s1 Set[T], s2 Set[T]) bool {
	return maps.Equal(s1, s2)
}
