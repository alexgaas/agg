package main

import (
	"bufio"
	"dist-group/base"
	v1 "dist-group/base/hashmap/open_addressing/linear_probing/v1"
	"dist-group/base/hashmap/two_level"
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
					recordsResult <- records
					break
				}
			}
		}(src)
	}

	dataBlocks := make([][]base.Phone, 0)
	for i := 0; i < numbJobs; i++ {
		dataBlocks = append(dataBlocks, <-recordsResult)
	}

	// parallel aggregate phase
	hashTableAsResult := make(chan two_level.TwoLevelHashMap)

	for _, block := range dataBlocks {
		blockToRead := block
		go func() {
			twoLevelHashTable := new(two_level.TwoLevelHashMap).New()
			for _, phone := range blockToRead {
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
