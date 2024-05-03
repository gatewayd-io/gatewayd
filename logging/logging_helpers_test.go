package logging

import (
	"bufio"
	"log"
	"net"
)

func testServer(network, address string) {
	// Create a new listener
	listener, err := net.Listen(network, address)
	if err != nil {
		log.Println(err)
		return
	}

	// Accept incoming connections
	conn, err := listener.Accept()
	if err != nil {
		log.Println(err)
		return
	}

	// Handle the connection
	handleConnection(conn)
}

func handleConnection(conn net.Conn) {
	// Close the connection
	defer conn.Close()

	// Create a new scanner
	scanner := bufio.NewScanner(conn)

	// Scan the connection
	for scanner.Scan() {
		// Print the scanned text
		log.Println(scanner.Text())
	}
}
