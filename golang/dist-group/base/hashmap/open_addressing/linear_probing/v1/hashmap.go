package v1

import (
	"unsafe"
)

// Basic murmur finalizer - https://gist.github.com/dnbaker/0fc1d4edbbdb24069eb063dc2559e4f5
func murmurFinalizerHash64(hash uint64) uint64 {
	hash ^= hash >> 33
	hash *= 0xff51afd7ed558ccd
	hash ^= hash >> 33
	hash *= 0xc4ceb9fe1a85ec53
	hash ^= hash >> 33
	return hash
}

/*
HashTableWithLinearProbing implementation
*/
const (
	defaultCapacity int = 8
)

var length int
var size int

const (
	Null  = 0
	Value = 1
)

type Cell struct {
	Key   string
	Value int
	state int
}

type HashTableWithLinearProbing struct {
	Cells []Cell
}

func (hashMap *HashTableWithLinearProbing) New() *HashTableWithLinearProbing {
	return hashMap.hashMapWithCapacity(defaultCapacity)
}

func (hashMap *HashTableWithLinearProbing) hashMapWithCapacity(capacity int) *HashTableWithLinearProbing {
	size = 0
	length = capacity
	cells := make([]Cell, length)
	return &HashTableWithLinearProbing{Cells: cells}
}

func getCell(hash uint64) uint64 {
	return hash % uint64(length)
}

func linearProbing(cell uint64) uint64 {
	return (cell + 1) % uint64(length)
}

func HashStringKey(key string) uint64 {
	// very simple hash from string:
	// get uint64 value as start pointer of string in memory
	// will work only for non-empty strings
	data := []byte(key)
	toHash := *(*uint64)(unsafe.Pointer(&data[0]))
	return murmurFinalizerHash64(toHash)
}

func (hashMap *HashTableWithLinearProbing) resize(capacity int) {
	oldTable := hashMap.Cells
	*hashMap = *hashMap.hashMapWithCapacity(capacity)
	size = 0
	for _, cell := range oldTable {
		if &cell != nil && cell.state == Value {
			hashMap.Put(cell.Key, cell.Value)
		}
	}
}

func (hashMap *HashTableWithLinearProbing) Size() int {
	return size
}

func (hashMap *HashTableWithLinearProbing) ContainsKey(key string) bool {
	return hashMap.Get(key) != nil
}

func (hashMap *HashTableWithLinearProbing) Put(key string, value int) {
	if key == "" {
		return
	}

	hash := HashStringKey(key)
	cell := getCell(hash)
	startIdx := cell

	for &hashMap.Cells[cell] != nil && hashMap.Cells[cell].state != Null {
		// update value of cell if it exists
		if hashMap.Cells[cell].Key == key && hashMap.Cells[cell].state == Value {
			hashMap.Cells[cell].Value = value
			return
		}

		// make linear probing
		cell = linearProbing(cell)
		if cell == startIdx {
			hashMap.resize(length * 2)
			cell = getCell(hash)
			startIdx = cell
		}
	}

	hashMap.Cells[cell] = Cell{
		Key:   key,
		Value: value,
		state: Value,
	}
	size++
}

func (hashMap *HashTableWithLinearProbing) Get(key string) *Cell {
	if key == "" {
		return nil
	}

	hash := HashStringKey(key)
	cell := getCell(hash)
	startIdx := cell
	for &hashMap.Cells[cell] != nil && hashMap.Cells[cell].state != Null {
		if hashMap.Cells[cell].Key == key && hashMap.Cells[cell].state == Value {
			return &hashMap.Cells[cell]
		}
		cell = linearProbing(cell)
		if cell == startIdx {
			return nil
		}
	}
	return nil
}

func (hashMap *HashTableWithLinearProbing) Remove(key string) {
	if key == "" {
		return
	}

	hash := HashStringKey(key)
	cell := getCell(hash)
	startIdx := cell
	for &hashMap.Cells[cell] != nil {
		if hashMap.Cells[cell].Key == key && hashMap.Cells[cell].state == Value {
			hashMap.Cells[cell].state = Null
			size--
			break
		}
		cell = linearProbing(cell)
		if cell == startIdx {
			break
		}
	}
	if size == length/4 && length/2 != 0 {
		hashMap.resize(length / 2)
	}
}
