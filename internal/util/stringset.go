package util

type StringSet struct {
	set map[string]bool
}

func (s *StringSet) Add(v string) {
	if s.set == nil {
		s.set = make(map[string]bool)
	}
	s.set[v] = true
}

func (s *StringSet) ToArray() []string {
	var return_list []string
	for key, pres := range s.set {
		if pres {
			return_list = append(return_list, key)
		}
	}
	return return_list
}
