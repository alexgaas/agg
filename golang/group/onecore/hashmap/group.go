package hashmap

import (
	"group/base"
	"group/base/hashmap/open_addressing/linear_probing/v1"
	"log"
)

func GroupByOsAndSumByPopularity() {
	records := base.Data()
	hashTable := new(v1.HashTableWithLinearProbing).New()

	for idx, record := range records {
		// pass csv caption
		if idx == 0 {
			continue
		}

		phone := base.MapPhone(record)

		if hashTable.ContainsKey(phone.Os) {
			popularity := hashTable.Get(phone.Os).Value + phone.Popularity
			hashTable.Put(phone.Os, popularity)
		} else {
			hashTable.Put(phone.Os, phone.Popularity)
		}
	}

	// print out result
	for _, cell := range hashTable.Cells {
		if cell.Key != "" {
			log.Printf("Popularity %d for group %s", cell.Value, cell.Key)
		}
	}
	log.Println()
}
