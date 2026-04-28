package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:9090")
	if err != nil {
		fmt.Printf("dial failed: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	message := "hello tcp echo\n"

	if _, err := conn.Write([]byte(message)); err != nil {
		fmt.Printf("write failed: %v\n", err)
		os.Exit(1)
	}

	reply, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Printf("read failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("server replied: %s", reply)
}
