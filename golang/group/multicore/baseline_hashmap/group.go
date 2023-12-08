package baseline_hashmap

import (
	"group/base"
	"group/base/buffer"
	"group/base/hashmap/open_addressing/linear_probing/v1"
	"log"
	"runtime"
	"sync"
)

func GroupByOsAndSumByPopularity() {
	// use all cores on your machine
	runtime.GOMAXPROCS(runtime.NumCPU())

	GroupByWorkingPool(GroupByOsAndSumByPopularityWorkerFn)
}

func GroupByOsAndSumByPopularityWorkerFn(job [][]string) *v1.HashTableWithLinearProbing {
	hashMap := new(v1.HashTableWithLinearProbing).New()
	for idx, record := range job {
		// pass csv caption
		if idx == 0 {
			continue
		}
		phone := base.MapPhone(record)

		if hashMap.ContainsKey(phone.Os) {
			popularity := hashMap.Get(phone.Os).Value
			hashMap.Put(phone.Os, popularity+phone.Popularity)
		} else {
			hashMap.Put(phone.Os, phone.Popularity)
		}
	}
	return hashMap
}

func workerPool(
	jobs <-chan [][]string,
	results chan<- v1.HashTableWithLinearProbing,
	fnAggregate func(job [][]string) *v1.HashTableWithLinearProbing) {
	var wg sync.WaitGroup

	var jobNumber = 0
	for j := range jobs {
		wg.Add(1)
		// we start a goroutine to run the job
		go func(job [][]string, jobNumber int) {
			hashMap := fnAggregate(job)

			// for tracing purpose
			/*
				log.Printf("job number %d\n", jobNumber)
				for _, cell := range hashMap.Cells {
					if cell.Value > 0 {
						log.Printf("Popularity %d for group %s", cell.Value, cell.Key)
					}
				}
				log.Println()
			*/

			wg.Done()

			results <- *hashMap
		}(j, jobNumber)
		jobNumber++
	}

	wg.Wait()
}

func GroupByWorkingPool(fnAggregate func(job [][]string) *v1.HashTableWithLinearProbing) {
	runtime.GOMAXPROCS(runtime.NumCPU())

	results := base.Data()
	var numbJobs = runtime.NumCPU() / 2
	ratio := len(results) / numbJobs

	buf := buffer.DataBuffer{}

	// fill up buffer with data
	for _, result := range results {
		err := buf.WriteLine(result)
		if err != nil {
			return
		}
	}

	jobs := make(chan [][]string, numbJobs)
	hashTableAsResult := make(chan v1.HashTableWithLinearProbing)
	hashTables := make([]v1.HashTableWithLinearProbing, 0)

	// define jobs
	for j := 0; j < numbJobs; j++ {
		if j == numbJobs-1 {
			reminder := len(results) - ratio*numbJobs
			ratio = ratio + reminder
		}
		resultSlice := buf.Next(ratio)
		jobs <- resultSlice
	}

	go workerPool(jobs, hashTableAsResult, fnAggregate)

	close(jobs)

	// aggregation phase
	for r := 0; r < numbJobs; r++ {
		hashTables = append(hashTables, <-hashTableAsResult)
	}

	// merge phase
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

	close(hashTableAsResult)

	// print out result
	for _, cell := range hashTables[0].Cells {
		if cell.Key != "" {
			log.Printf("Popularity %d for group %s", cell.Value, cell.Key)
		}
	}
	log.Println()
}
