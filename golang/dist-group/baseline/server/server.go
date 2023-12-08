package main

import (
	"bufio"
	"dist-group/base"
	v1 "dist-group/base/hashmap/open_addressing/linear_probing/v1"
	v2 "dist-group/base/hashmap/open_addressing/linear_probing/v2"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strings"
)

var addr = flag.String("addr", "", "The address to listen to; default is \"\" (all interfaces).")
var ports = flag.String("ports", "8001, 8002, 8003, 8004", "Ports to listen on; defaults are [8001, 8002, 8003, 8004].")

func main() {
	// use all cores on your machine
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()

	fmt.Println("Starting server...")

	globalHashMap := new(v2.HashTableWithLinearProbing).New()
	hashTableAsResult := make(chan v1.HashTableWithLinearProbing)

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

			localHashMap := new(v1.HashTableWithLinearProbing).New()

			done := make(chan struct{})

			for {
				conn, err := listener.Accept()
				if err != nil {
					fmt.Printf("Some connection error: %s\n", err)
				}

				// read data
				go handleConnection(conn, globalHashMap, localHashMap, done)

				select {
				case <-done:
					// all data been read from channel
					hashTableAsResult <- *localHashMap
					break
				}
			}
		}(src)
	}

	hashTables := make([]v1.HashTableWithLinearProbing, 0)
	for i := 0; i < numbJobs; i++ {
		hashTables = append(hashTables, <-hashTableAsResult)
		// for debugging purpose
		/*
			select {
			case msg := <-hashTableAsResult:
				fmt.Printf("got %v from worker channel\n", msg)
			}
		*/
	}

	// print out result
	for _, cell := range globalHashMap.Cells {
		if cell.Key != "" {
			log.Printf("Popularity %d for group %s", cell.Value, cell.Key)
		}
	}
	log.Println()
}

func handleConnection(conn net.Conn, globalHashMap *v2.HashTableWithLinearProbing, localHashMap *v1.HashTableWithLinearProbing, done chan struct{}) {
	remoteAddr := conn.RemoteAddr().String()
	fmt.Println("Client connected from " + remoteAddr)

	scanner := bufio.NewScanner(conn)

	for {
		ok := scanner.Scan()

		if !ok {
			break
		}

		handleMessage(scanner.Text(), conn, globalHashMap, localHashMap)
	}

	fmt.Println("Client at " + remoteAddr + " disconnected.")

	done <- struct{}{}
}

func handleMessage(message string, conn net.Conn, globalHashMap *v2.HashTableWithLinearProbing, localHashMap *v1.HashTableWithLinearProbing) {
	// for debugging purpose
	// fmt.Println("> " + message)

	onExit(message, conn)

	phone := base.MapPhone(strings.Split(message, ","))
	if phone.Os == "" {
		return
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
