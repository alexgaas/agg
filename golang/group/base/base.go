package base

import (
	"bufio"
	"encoding/csv"
	"log"
	"os"
	"strconv"
)

type Phone struct {
	id            int
	BrandName     string
	modelName     string
	Os            string
	Popularity    int
	bestPrice     float64
	lowestPrice   float64
	highestPrice  float64
	sellersAmount int
	screenSize    float64
	memorySize    float64
	batterySize   float64
	releaseDate   string
	BucketId      int
}

type GroupByBrandPhone struct {
	BrandName  string
	Popularity int
}

type GroupByOsPhone struct {
	Os         string
	Popularity int
}

func MapPhone(record []string) Phone {
	id, _ := strconv.Atoi(record[0])
	popularity, _ := strconv.Atoi(record[4])
	bestPrice, _ := strconv.ParseFloat(record[5], 1)
	lowestPrice, _ := strconv.ParseFloat(record[6], 1)
	highestPrice, _ := strconv.ParseFloat(record[7], 1)
	sellersAmount, _ := strconv.Atoi(record[8])
	screenSize, _ := strconv.ParseFloat(record[9], 1)
	memorySize, _ := strconv.ParseFloat(record[10], 1)
	batterySize, _ := strconv.ParseFloat(record[11], 1)
	bucketId, _ := strconv.Atoi(record[13])
	return Phone{
		id:            id,
		BrandName:     record[1],
		modelName:     record[2],
		Os:            record[3],
		Popularity:    popularity,
		bestPrice:     bestPrice,
		lowestPrice:   lowestPrice,
		highestPrice:  highestPrice,
		sellersAmount: sellersAmount,
		screenSize:    screenSize,
		memorySize:    memorySize,
		batterySize:   batterySize,
		releaseDate:   record[12],
		BucketId:      bucketId,
	}
}

func Data() [][]string {
	file, fileErr := os.Open("../../base/test/phones_data.csv")
	if fileErr != nil {
		log.Fatalln(fileErr)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	reader := bufio.NewReader(file)

	csvReader := csv.NewReader(reader)

	results, csvReadErr := csvReader.ReadAll()
	if csvReadErr != nil {
		log.Fatalln(fileErr)
	}

	return results
}
