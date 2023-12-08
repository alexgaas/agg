package main

import (
	"bufio"
	"dist-group/base"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strings"
)

var addr = flag.String("addr", "", "The address to listen to; default is \"\" (all interfaces).")
var ports = flag.String("ports", "8001,8002,8003,8004", "Ports to listen on; defaults are [8001, 8002, 8003, 8004].")

func main() {
	// use all cores on your machine
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()

	fmt.Println("Starting server...")

	recordsResult := make(chan []base.Phone)

	numbJobs := len(strings.Split(*ports, ","))
	for i := 0; i < numbJobs; i++ {
		src := *addr + ":" + strings.Split(*ports, ",")[i]
		go func(source string) {
			listener, _ := net.Listen("tcp", src)
			fmt.Printf("Listening on %s.\n", src)

			defer func(listener net.Listener) {
				err := listener.Close()
				if err != nil {
					fmt.Printf("Fatal connection error: %s\n", err)
					os.Exit(-1)
				}
			}(listener)

			records := make([]base.Phone, 0)

			done := make(chan struct{})

			for {
				conn, err := listener.Accept()
				if err != nil {
					fmt.Printf("Some connection error: %s\n", err)
				}

				// read data
				go handleConnection(conn, &records, done)

				select {
				case <-done:
					// all data been read from channel

					// TODO - instead reading all data from channel we can make sort on after reading only N elements
					// from channel. That will allow us to have O(1) memory but will increase runtime since we have
					// merge in heap every time we get N records. We also will need synchronization between goroutines
					recordsResult <- records
					break
				}
			}
		}(src)
	}

	results := make([]base.Phone, 0)
	for i := 0; i < numbJobs; i++ {
		results = append(results, <-recordsResult...)
	}

	// parallel sort of results
	sortedResult := make(chan []base.Phone)
	go MergeSort(results, sortedResult)
	r := <-sortedResult

	printResults(r)
}

func handleConnection(conn net.Conn, records *[]base.Phone, done chan struct{}) {
	remoteAddr := conn.RemoteAddr().String()
	fmt.Println("Client connected from " + remoteAddr)

	scanner := bufio.NewScanner(conn)

	for {
		ok := scanner.Scan()

		if !ok {
			break
		}

		handleMessage(scanner.Text(), conn, records)
	}

	fmt.Println("Client at " + remoteAddr + " disconnected.")

	done <- struct{}{}
}

func handleMessage(message string, conn net.Conn, records *[]base.Phone) {
	// for debugging purpose
	// fmt.Println("> " + message)
	*records = append(*records, base.MapPhone(strings.Split(message, ",")))

	onExit(message, conn)
}

func onExit(message string, conn net.Conn) {
	if len(message) > 0 && message[0] == '/' {
		switch {
		case message == "/quit":
			fmt.Println("Quitting.")
			_, _ = conn.Write([]byte("I'm shutting down now.\n"))
			fmt.Println("< " + "%quit%")
			_, _ = conn.Write([]byte("%quit%\n"))
			os.Exit(0)
		default:
			_, _ = conn.Write([]byte("Unrecognized command.\n"))
		}
	}
}

func printResults(phones []base.Phone) {
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

func Merge(leftData []base.Phone, rightData []base.Phone) (result []base.Phone) {
	result = make([]base.Phone, len(leftData)+len(rightData))
	lid, rid := 0, 0

	for i := 0; i < cap(result); i++ {
		switch {
		case lid >= len(leftData):
			result[i] = rightData[rid]
			rid++
		case rid >= len(rightData):
			result[i] = leftData[lid]
			lid++
		case leftData[lid].Os < rightData[rid].Os:
			result[i] = leftData[lid]
			lid++
		default:
			result[i] = rightData[rid]
			rid++
		}
	}

	return
}

func MergeSort(data []base.Phone, r chan []base.Phone) {
	if len(data) == 1 {
		r <- data
		return
	}

	leftChan := make(chan []base.Phone)
	rightChan := make(chan []base.Phone)
	middle := len(data) / 2

	go MergeSort(data[:middle], leftChan)
	go MergeSort(data[middle:], rightChan)

	leftData := <-leftChan
	rightData := <-rightChan

	close(leftChan)
	close(rightChan)
	r <- Merge(leftData, rightData)
	return
}
