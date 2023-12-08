package simple_array

import (
	"group/base"
	"log"
	"sort"
)

func GroupByOsAndSumByPopularity() {
	phones := PrepareData()

	// group by Brand name
	sort.Slice(phones, func(i, j int) bool {
		return phones[i].Os > phones[j].Os
	})

	var groupByOsAndSumByPopularity []base.GroupByOsPhone
	var currentOsName string
	var currentPopularity int
	for idx, record := range phones {
		if idx == 0 {
			currentOsName = record.Os
		}

		if record.Os == "" {
			continue
		}

		if currentOsName != record.Os {
			groupByOsAndSumByPopularity = append(groupByOsAndSumByPopularity, base.GroupByOsPhone{
				Os:         currentOsName,
				Popularity: currentPopularity,
			})
			currentPopularity = 0
		}
		currentOsName = record.Os
		currentPopularity += record.Popularity
	}
	// insert last group
	groupByOsAndSumByPopularity = append(groupByOsAndSumByPopularity, base.GroupByOsPhone{
		Os:         currentOsName,
		Popularity: currentPopularity,
	})

	// print out result
	for _, group := range groupByOsAndSumByPopularity {
		log.Printf("Popularity %d for group %s", group.Popularity, group.Os)
	}
	log.Println()
}

func PrepareData() []base.Phone {
	// Prepare data for group by
	records := base.Data()
	var phones []base.Phone

	for idx, record := range records {
		// pass csv caption
		if idx == 0 {
			continue
		}
		phones = append(phones, base.MapPhone(record))
	}
	return phones
}
