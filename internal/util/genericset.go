package util

type Set[T comparable] struct {
	set map[T]bool
}

func (s *Set[comparable]) Has(check comparable) bool {
	if s.set == nil {
		s.set = make(map[comparable]bool)
	}
	shouldInclude, present := s.set[check]
	return present && shouldInclude
}

func (s *Set[comparable]) Add(value comparable) {
	if s.set == nil {
		s.set = make(map[comparable]bool)
	}
	s.set[value] = true
}

func (s *Set[comparable]) Remove(value comparable) {
	if s.set == nil {
		s.set = make(map[comparable]bool)
	}
	s.set[value] = false
}

func (s *Set[comparable]) ToArray() []comparable {
	var return_list []comparable
	for key, pres := range s.set {
		if pres {
			return_list = append(return_list, key)
		}
	}
	return return_list
}
