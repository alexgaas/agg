package main

import (
	"bufio"
	"flag"
	"log"
	"net"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"time"
)

var host = flag.String("host", "localhost", "The hostname or IP to connect to; defaults to \"localhost\".")
var port = flag.Int("port", 8001, "The port to connect to; defaults to 8001.")
var filePath = flag.String("file", "/Users/alex.gaas/Desktop/go/dist-group/base/data/phones_data.csv", "File we send for aggregation on the server initiator.")

func main() {
	// use all cores on your machine
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()

	dest := *host + ":" + strconv.Itoa(*port)
	log.Printf("Connecting to %s...\n", dest)

	// establish connection
	conn, err := net.Dial("tcp", dest)

	if err != nil {
		if _, t := err.(*net.OpError); t {
			log.Println("Some problem connecting.")
		} else {
			log.Println("Unknown error: " + err.Error())
		}
		os.Exit(1)
	}

	// read commands from server
	go readConnection(conn)

	file, fileErr := os.Open(*filePath)
	if fileErr != nil {
		log.Fatalln(fileErr)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	reader := bufio.NewReader(file)

	isHeader := false

	log.Printf("Sending data to %s...\n", dest)

	for {
		// simply read by EOL
		text, _ := reader.ReadBytes('\n')
		// pass first line, that's header
		if !isHeader {
			isHeader = true
			continue
		}

		// set deadline
		_ = conn.SetWriteDeadline(time.Now().Add(1 * time.Second))

		// write data to aggregate to server line by line
		_, err := conn.Write(text)
		if err != nil {
			log.Println("Error writing to stream.")
			break
		}

		// when whole been read leave
		if len(text) == 0 {
			log.Println("Reached EOF on server connection.")
			break
		}
	}
}

func readConnection(conn net.Conn) {
	for {
		scanner := bufio.NewScanner(conn)

		for {
			ok := scanner.Scan()
			text := scanner.Text()

			command := handleCommands(text)
			if !command {
				log.Printf("\b\b** %s\n> ", text)
			}

			if !ok {
				log.Println("Reached EOF on server connection.")
				break
			}
		}
	}
}

func handleCommands(text string) bool {
	r, err := regexp.Compile("^%.*%$")
	if err == nil {
		if r.MatchString(text) {

			switch {
			case text == "%quit%":
				log.Println("\b\bServer is leaving. Hanging up.")
				os.Exit(0)
			}

			return true
		}
	}

	return false
}
