package hashmap

import v1 "dist-group/base/hashmap/open_addressing/linear_probing/v1"

const BitsForBucket = 8
const NumBuckets = 1 << BitsForBucket
const MaxBucket = NumBuckets - 1

func getBucketFromHash(hashValue int) int {
	return hashValue >> (32 - BitsForBucket) & MaxBucket
}

func GetBucket(key string) int {
	hash := v1.HashStringKey(key)
	bucket := getBucketFromHash(int(hash))
	return bucket
}
