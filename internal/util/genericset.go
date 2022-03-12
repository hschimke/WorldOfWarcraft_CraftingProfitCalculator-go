package util

// Set is a true set of items
type Set[T comparable] struct {
	set map[T]bool
}

// Has checks if a Set contains an element
func (s *Set[comparable]) Has(check comparable) bool {
	if s.set == nil {
		s.set = make(map[comparable]bool)
	}
	shouldInclude, present := s.set[check]
	return present && shouldInclude
}

// Add adds an element to a Set
func (s *Set[comparable]) Add(value comparable) {
	if s.set == nil {
		s.set = make(map[comparable]bool)
	}
	s.set[value] = true
}

// Remove drops an element from a Set
func (s *Set[comparable]) Remove(value comparable) {
	if s.set == nil {
		s.set = make(map[comparable]bool)
	}
	s.set[value] = false
}

// ToSlice converts a set into a slice
func (s *Set[comparable]) ToSlice() []comparable {
	var return_list []comparable
	for key, pres := range s.set {
		if pres {
			return_list = append(return_list, key)
		}
	}
	return return_list
}

// SetFromSlice takes a slice and returns a Set
func SetFromSlice[T comparable](source []T) *Set[T] {
	var set Set[T]
	for _, val := range source {
		set.Add(val)
	}
	return &set
}

// SetEqual compares to Sets
func SetEqual[T comparable](s1 Set[T], s2 Set[T]) bool {
	t1 := s1.ToSlice()
	t2 := s2.ToSlice()

	if len(t1) != len(t2) {
		return false
	}
	found := true
	for element, value := range s1.set {
		if value {
			found = found && s2.Has(element)
		}
	}
	return found
}
