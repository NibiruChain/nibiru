package keys

func Join[K1 Key, K2 Key](k1 K1, k2 K2) Two[K1, K2] {
	return Two[K1, K2]{
		k1: k1,
		k2: k2,
	}
}

func SubPrefix[K1 Key, K2 Key](k1 K1) Two[K1, K2] {
	panic("")
}

type Two[K1 Key, K2 Key] struct {
	k1 K1
	k2 K2
}

func (t Two[K1, K2]) FromPrimaryKeyBytes(b []byte) Key {
	panic("impl")
}

func (t Two[K1, K2]) PrimaryKey() []byte {
	return append(t.k1.PrimaryKey(), t.k2.PrimaryKey()...)
}

func (t Two[K1, K2]) SecondaryKey() []byte {
	return append(t.k1.SecondaryKey(), t.k2.SecondaryKey()...)
}
