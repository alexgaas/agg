package parititioning

import (
	"group/base"
	"group/base/buffer"
	"group/base/hashmap/open_addressing/linear_probing/v1"
	"log"
	"runtime"
	"strconv"
	"sync"
	"unsafe"
)

func GroupByOsAndSumByPopularity() {
	// use all cores on your machine
	runtime.GOMAXPROCS(runtime.NumCPU())

	// very simple explanation of bucket placement algorithm
	// simpleExampleOfTasksToBucket

	GroupByDataBlocks()
}

func GroupByDataBlocks() {
	// define bucket size, please see explanation inside
	var numBuckets = buffer.DefineBucketSize()

	// prepare data
	records := base.Data()
	dataBlocks, err := buffer.MakePartitioning(records)
	if err != nil {
		log.Fatalln(err)
	}

	/*
		Phase 1 - mark data blocks to buckets in parallel
	*/
	var wg sync.WaitGroup

	var bucketToGroupMap, bucketToTaskMap sync.Map
	taskNumber := 0
	for _, block := range dataBlocks {
		blockToRead := block.Read()
		wg.Add(1)
		// mark buckets in blocks
		go func() {
			for _, record := range blockToRead {
				key := record[3]
				if key == "" {
					continue
				}

				// simple hash to make bucket as `hash: key -> bucket_num`
				data := []byte(key)
				toHash := *(*uint64)(unsafe.Pointer(&data[0]))
				bucketId := hash(toHash, numBuckets)

				// mark bucket number in records
				record[13] = strconv.Itoa(bucketId)

				_, ok := bucketToGroupMap.Load(bucketId)
				if !ok {
					bucketToGroupMap.Store(bucketId, key)
				}

				_, ok = bucketToTaskMap.Load(bucketId)
				if !ok {
					bucketToTaskMap.Store(bucketId, taskNumber)
					taskNumber++
				}
			}
			wg.Done()
		}()
	}

	wg.Wait()

	/*
		Phase 2 - aggregate by bucket number in parallel
	*/
	aggregateChannel := make(chan int)
	tasksToAggregation := new(v1.HashTableWithLinearProbing).New()

	for taskId := 0; taskId < lenSyncMap(&bucketToGroupMap); taskId++ {
		go func(task int) {
			for _, block := range dataBlocks {
				blockToRead := block.Read()

				for _, record := range blockToRead {

					phone := base.MapPhone(record)

					if phone.Os == "" || phone.BucketId == 0 {
						continue
					}

					jobNumber, _ := bucketToTaskMap.Load(phone.BucketId)
					if task != jobNumber {
						continue
					}

					group, ok := bucketToGroupMap.Load(phone.BucketId)
					if ok {
						value := phone.Popularity
						if tasksToAggregation.ContainsKey(group.(string)) {
							value = value + tasksToAggregation.Get(group.(string)).Value
						}

						tasksToAggregation.Put(group.(string), value)
					}
				}
			}
			aggregateChannel <- task
		}(taskId)
	}

	for taskId := 0; taskId < lenSyncMap(&bucketToGroupMap); taskId++ {
		<-aggregateChannel
		// for debugging purpose
		/*
			select {
			case msg := <-aggregateChannel:
				fmt.Printf("got %v from worker channel\n", msg)
			}
		*/
	}

	close(aggregateChannel)

	// print out result
	for _, cell := range tasksToAggregation.Cells {
		if cell.Key != "" {
			log.Printf("Popularity %d for group %s", cell.Value, cell.Key)
		}
	}
	log.Println()
}

func lenSyncMap(m *sync.Map) int {
	var i int
	m.Range(func(k, v interface{}) bool {
		i++
		return true
	})
	return i
}

func simpleExampleOfTasksToBucket() {
	var numBuckets = 20
	var numTasks = 40

	var tasks []int

	for taskId := 0; taskId < numTasks; taskId++ {
		tasks = append(tasks, taskId)
	}

	bucketToTasks := make(map[int][]int)
	for _, task := range tasks {
		bucketId := hash(uint64(task), numBuckets)
		_, ok := bucketToTasks[bucketId]
		if !ok {
			bucketToTasks[bucketId] = make([]int, 0)
		}
		val, ok := bucketToTasks[bucketId]
		if ok {
			bucketToTasks[bucketId] = append(val, task)
		}
	}

	for i := 0; i < numBuckets; i++ {
		log.Printf("Bucket: %d, Tasks: %v", i, bucketToTasks[i])
	}
}

func hash(key uint64, task int) int {
	hash := key

	// hash *= 0xff51afd7ed558ccd
	// hash ^= hash >> 33

	// multiplicative inverse - https://stackoverflow.com/questions/664014/what-integer-hash-function-are-good-that-accepts-an-integer-hash-key
	hash = ((hash >> 16) ^ hash) * 0x45d9f3b
	hash = ((hash >> 16) ^ hash) * 0x45d9f3b
	hash = (hash >> 16) ^ hash

	return int(hash % uint64(task+1))
}
