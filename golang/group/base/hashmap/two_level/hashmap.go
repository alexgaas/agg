package two_level

import (
	v1 "group/base/hashmap/open_addressing/linear_probing/v1"
)

const BitsForBucket = 8
const NumBuckets = 1 << BitsForBucket
const MaxBucket = NumBuckets - 1

type TwoLevelHashMap struct {
	Buckets []*v1.HashTableWithLinearProbing
}

func (hashMap *TwoLevelHashMap) New() *TwoLevelHashMap {
	return hashMap.hashMapWithCapacity()
}

func (hashMap *TwoLevelHashMap) hashMapWithCapacity() *TwoLevelHashMap {
	buckets := make([]*v1.HashTableWithLinearProbing, NumBuckets)
	return &TwoLevelHashMap{Buckets: buckets}
}

func getBucketFromHash(hashValue int) int {
	return hashValue >> (32 - BitsForBucket) & MaxBucket
}

func getBucket(key string, hashMap *TwoLevelHashMap) int {
	hash := v1.HashStringKey(key)
	bucket := getBucketFromHash(int(hash))
	if hashMap.Buckets[bucket] == nil {
		hashMap.Buckets[bucket] = new(v1.HashTableWithLinearProbing).New()
	}
	return bucket
}

func (hashMap *TwoLevelHashMap) Put(key string, value int) {
	if key == "" {
		return
	}

	bucket := getBucket(key, hashMap)
	hashMap.Buckets[bucket].Put(key, value)
}

func (hashMap *TwoLevelHashMap) Get(key string) *v1.Cell {
	if key == "" {
		return nil
	}

	hash := v1.HashStringKey(key)
	bucket := getBucketFromHash(int(hash))
	if hashMap.Buckets[bucket] == nil {
		hashMap.Buckets[bucket] = new(v1.HashTableWithLinearProbing).New()
	}
	return hashMap.Buckets[bucket].Get(key)
}
