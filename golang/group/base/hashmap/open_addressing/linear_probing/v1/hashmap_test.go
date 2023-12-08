package v1

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHashMapLinearProbing(t *testing.T) {
	hashTable := new(HashTableWithLinearProbing).New()

	hashTable.Put("1", 1)
	hashTable.Put("2", 2)
	hashTable.Put("3", 3)
	hashTable.Put("4", 4)

	hashTable.Put("5", 5)
	hashTable.Put("6", 6)
	hashTable.Put("7", 7)
	hashTable.Put("8", 8)

	hashTable.Put("9", 9)
	hashTable.Put("10", 10)
	hashTable.Put("11", 11)
	hashTable.Put("12", 12)

	require.NotEmpty(t, hashTable)

	require.Exactly(t, hashTable.Get("1").Value, 1)
	require.Exactly(t, hashTable.Get("2").Value, 2)
	require.Exactly(t, hashTable.Get("3").Value, 3)
	require.Exactly(t, hashTable.Get("4").Value, 4)
	require.Exactly(t, hashTable.Get("5").Value, 5)
	require.Exactly(t, hashTable.Get("6").Value, 6)
	require.Exactly(t, hashTable.Get("7").Value, 7)
	require.Exactly(t, hashTable.Get("8").Value, 8)
	require.Exactly(t, hashTable.Get("9").Value, 9)
	require.Exactly(t, hashTable.Get("10").Value, 10)
	require.Exactly(t, hashTable.Get("11").Value, 11)
	require.Exactly(t, hashTable.Get("12").Value, 12)

	require.True(t, hashTable.Get("test") == nil)

	hashTable.Remove("9")
	require.True(t, hashTable.Get("9") == nil)

	hashTable.Remove("10")
	hashTable.Remove("11")
	hashTable.Remove("12")

	require.True(t, hashTable.Size() == 8)
}
