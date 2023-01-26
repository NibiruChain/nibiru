package set

type Set[T comparable] map[T]struct{}

func (set Set[T]) Add(s T) {
	set[s] = struct{}{}
}

func (set Set[T]) Remove(s T) {
	delete(set, s)
}

func (set Set[T]) Has(s T) bool {
	_, ok := set[s]
	return ok
}

func (set Set[T]) Len() int {
	return len(set.ToSlice())
}

func (set Set[T]) ToSlice() []T {
	var slice []T
	for s := range set {
		slice = append(slice, s)
	}
	return slice
}

func New[T comparable](strs ...T) Set[T] {
	set := Set[T]{}
	for _, s := range strs {
		set.Add(s)
	}
	return set
}
