package common

type StringSet map[string]struct{}

func (set StringSet) Add(s string) {
	set[s] = struct{}{}
}

func (set StringSet) Remove(s string) {
	delete(set, s)
}

func (set StringSet) Has(s string) bool {
	_, ok := set[s]
	return ok
}

func (set StringSet) Len() int {
	return len(set.ToSlice())
}

func (set StringSet) ToSlice() []string {
	var slice []string
	for s := range set {
		slice = append(slice, s)
	}
	return slice
}

func NewStringSet(strs ...string) StringSet {
	set := StringSet{}
	for _, s := range strs {
		set.Add(s)
	}
	return set
}
