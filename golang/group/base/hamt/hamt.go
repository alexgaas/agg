package hamt

func hash_str(data string) uint64 {
	var prime uint64 = 0x100000001b3
	var hash uint64 = 0xcbf29ce484222325
	for i := range data {
		hash ^= uint64(data[i])
		hash *= prime
	}
	return hash
}

type Hamt struct {
	children []*Hamt
	key      string
	value    int
}

func hamt_find_indirect(root *Hamt, key string) **Hamt {
	hash := hash_str(key)
	hamt := &root

	for hamt != nil {
		hamt = &(*hamt).children[hash&1]
		if *hamt == nil || (*hamt).key == key {
			break
		}
		hash >>= 1
	}

	return hamt
}

func hamt_find(root *Hamt, key string) *Hamt {
	return *hamt_find_indirect(root, key)
}

func hamt_add(root *Hamt, add *Hamt) {
	var found **Hamt = hamt_find_indirect(root, add.key)
	if *found == nil {
		*found = add
	}
}
