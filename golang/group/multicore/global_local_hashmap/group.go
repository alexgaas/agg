package global_local_hashmap

import (
	"group/base"
	"group/base/buffer"
	v1 "group/base/hashmap/open_addressing/linear_probing/v1"
	v2 "group/base/hashmap/open_addressing/linear_probing/v2"
	"log"
	"runtime"
)

func GroupByOsAndSumByPopularity() {
	// use all cores on your machine
	runtime.GOMAXPROCS(runtime.NumCPU())

	GroupByThreads()
}

func GroupByThreads() {
	// prepare data
	records := base.Data()
	dataBlocks, err := buffer.MakePartitioning(records)
	if err != nil {
		log.Fatalln(err)
	}

	hashTableAsResult := make(chan v1.HashTableWithLinearProbing)
	globalHashMap := new(v2.HashTableWithLinearProbing).New()

	for _, block := range dataBlocks {
		blockToRead := block.Read()
		// start thread-local table
		go func() {
			localHashMap := new(v1.HashTableWithLinearProbing).New()
			for _, record := range blockToRead {

				phone := base.MapPhone(record)
				if phone.Os == "" {
					continue
				}

				if localHashMap.ContainsKey(phone.Os) {
					localHashMap.Put(phone.Os, localHashMap.Get(phone.Os).Value+phone.Popularity)
				} else {
					globalPopularity := phone.Popularity
					if globalHashMap.Get(phone.Os) != nil {
						globalPopularity += globalHashMap.Get(phone.Os).Value
					}
					if globalHashMap.Put(phone.Os, globalPopularity) == v2.BreakerOpened {
						localPopularity := phone.Popularity
						if localHashMap.Get(phone.Os) != nil {
							localPopularity += localHashMap.Get(phone.Os).Value
						}
						localHashMap.Put(phone.Os, localPopularity)
					}
				}
			}
			hashTableAsResult <- *localHashMap
		}()
	}

	hashTables := make([]v1.HashTableWithLinearProbing, 0)
	for taskId := 0; taskId < len(dataBlocks); taskId++ {
		hashTables = append(hashTables, <-hashTableAsResult)
		// for debugging purpose
		/*
			select {
			case msg := <-hashTableAsResult:
				fmt.Printf("got %v from worker channel\n", msg)
			}
		*/
	}

	// merge phase
	for idx, table := range hashTables {
		// merge with first table
		if idx == 0 {
			continue
		}
		primaryTable := globalHashMap
		for _, cell := range table.Cells {
			primaryValue := cell.Value
			if cell.Key != "" && primaryTable.ContainsKey(cell.Key) {
				primaryCell := primaryTable.Get(cell.Key)
				primaryValue = primaryCell.Value + cell.Value
			}
			primaryTable.Put(cell.Key, primaryValue)
		}
	}

	close(hashTableAsResult)

	// print out result
	for _, cell := range globalHashMap.Cells {
		if cell.Key != "" {
			log.Printf("Popularity %d for group %s", cell.Value, cell.Key)
		}
	}
	log.Println()
}
