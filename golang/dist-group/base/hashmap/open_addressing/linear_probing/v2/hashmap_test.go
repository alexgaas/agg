package v2

import (
	"testing"
)

func TestHashMapAtomicBreaker(t *testing.T) {
	/*
		Setup breakpoint on [if] to validate we return breaker as opened in contention case
			// if breaker is opened
			if hashMap.Cells[cell].cellBreaker.Load() == true {
				return BreakerOpened
			}
	*/
	hashTable := new(HashTableWithLinearProbing).New()
	for i := 0; i < 64; i++ {
		go func() {
			hashTable.Put("1", 1)
			hashTable.Put("2", 2)
			hashTable.Put("3", 3)
			hashTable.Put("4", 4)

			hashTable.Put("5", 5)
			hashTable.Put("6", 6)
			hashTable.Put("7", 7)
			hashTable.Put("8", 8)
		}()
	}
}
