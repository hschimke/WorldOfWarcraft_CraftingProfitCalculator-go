package util

type UintSet struct {
	internal_map map[uint]bool
}

func (s *UintSet) Has(check uint) bool {
	if s.internal_map == nil {
		s.internal_map = make(map[uint]bool)
	}
	shouldInclude, present := s.internal_map[check]
	return present && shouldInclude
}

func (s *UintSet) Add(value uint) {
	if s.internal_map == nil {
		s.internal_map = make(map[uint]bool)
	}
	s.internal_map[value] = true
}
