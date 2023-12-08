package two_level_hashmap

import (
	"group/base"
	"group/base/buffer"
	v1 "group/base/hashmap/open_addressing/linear_probing/v1"
	"group/base/hashmap/two_level"
	"log"
	"runtime"
)

func GroupByOsAndSumByPopularity() {
	// use all cores on your machine
	runtime.GOMAXPROCS(runtime.NumCPU())

	// prepare data
	records := base.Data()
	dataBlocks, err := buffer.MakePartitioning(records)
	if err != nil {
		log.Fatalln(err)
	}

	hashTableAsResult := make(chan two_level.TwoLevelHashMap)

	for _, block := range dataBlocks {
		blockToRead := block.Read()
		go func() {
			twoLevelHashTable := new(two_level.TwoLevelHashMap).New()
			for _, record := range blockToRead {
				phone := base.MapPhone(record)
				if phone.Os == "" {
					continue
				}

				popularity := phone.Popularity
				hashTableCell := twoLevelHashTable.Get(phone.Os)
				if hashTableCell != nil {
					popularity += hashTableCell.Value
				}
				twoLevelHashTable.Put(phone.Os, popularity)
			}
			hashTableAsResult <- *twoLevelHashTable
		}()
	}

	twoLevelHashMaps := make([]two_level.TwoLevelHashMap, 0)
	for taskId := 0; taskId < len(dataBlocks); taskId++ {
		twoLevelHashMaps = append(twoLevelHashMaps, <-hashTableAsResult)
		// for debugging purpose
		/*
			select {
			case msg := <-hashTableAsResult:
				fmt.Printf("got %v from worker channel\n", msg)
			}
		*/
	}

	// merge phase - we shift data between buckets to aggregate in parallel
	result := make(chan bool)
	twoLevelHashTableOut := new(two_level.TwoLevelHashMap).New()

	for i := 0; i < two_level.NumBuckets; i++ {
		go func(bucketId int) {
			hashTables := make([]v1.HashTableWithLinearProbing, 0)
			for _, twoLevelHashTable := range twoLevelHashMaps {
				if twoLevelHashTable.Buckets[bucketId] == nil {
					continue
				}
				hashTables = append(hashTables, *twoLevelHashTable.Buckets[bucketId])
			}

			// internal merge phase of bucket hash maps
			for idx, table := range hashTables {
				// merge with first table
				if idx == 0 {
					continue
				}
				primaryTable := hashTables[0]
				for _, cell := range table.Cells {
					primaryValue := cell.Value
					if cell.Key != "" && primaryTable.ContainsKey(cell.Key) {
						primaryCell := primaryTable.Get(cell.Key)
						primaryValue = primaryCell.Value + cell.Value
					}
					primaryTable.Put(cell.Key, primaryValue)
				}
			}

			if len(hashTables) > 0 && &hashTables[0] != nil {
				twoLevelHashTableOut.Buckets[bucketId] = &hashTables[0]
			}

			result <- true
		}(i)
	}

	for i := 0; i < two_level.NumBuckets; i++ {
		<-result
		// for debugging purpose
		/*
			select {
			case msg := <-result:
				fmt.Printf("got %v from worker channel\n", msg)
			}
		*/
	}

	for _, hashMap := range twoLevelHashTableOut.Buckets {
		if hashMap == nil {
			continue
		}

		for _, cell := range hashMap.Cells {
			if cell.Key != "" {
				log.Printf("Popularity %d for group %s", cell.Value, cell.Key)
			}
		}
	}
	log.Println()
}
