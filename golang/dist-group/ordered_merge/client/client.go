package main

import (
	"bufio"
	"dist-group/base"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"
)

var host = flag.String("host", "localhost", "The hostname or IP to connect to; defaults to \"localhost\".")
var port = flag.Int("port", 8001, "The port to connect to; defaults to 8001.")
var filePath = flag.String("file", "/Users/alex.gaas/Desktop/go/dist-group/base/data/phones_data.csv", "File we send for aggregation on the server initiator.")

func main() {
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

	records := base.Data(*filePath)
	var phones []base.Phone
	for idx, record := range records {
		// pass csv caption
		if idx == 0 {
			continue
		}
		phones = append(phones, base.MapPhone(record))
	}

	// SORT data locally on the data node
	// We actually do not need to sort out here, that just will help to provide better runtime on server-initiator
	sort.Slice(phones, func(i, j int) bool {
		return phones[i].Os > phones[j].Os
	})

	// add temp file for sorted data
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("%s-", filepath.Base(os.Args[0])))
	if err != nil {
		log.Fatal("Could not create temporary file", err)
	}

	defer func(tmpFile *os.File) {
		err := tmpFile.Close()
		if err != nil {
			log.Fatal("Could not close temporary file", err)
		}

	}(tmpFile)
	for _, phone := range phones {
		message := base.MapRecord(phone)
		_, err := tmpFile.WriteString(message + "\n")
		if err != nil {
			log.Println("Error writing to temp file.")
			return
		}
	}

	_, err = tmpFile.Seek(0, 0)
	if err != nil {
		return
	}
	reader := bufio.NewReader(tmpFile)

	log.Printf("Sending data to %s...\n", dest)

	for {
		// simply read by EOL
		text, _ := reader.ReadBytes('\n')

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
