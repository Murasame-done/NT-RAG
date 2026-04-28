package main

import (
	"io"
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}
	defer listener.Close()

	log.Println("tcp echo server listening on :9090")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("accept failed: %v", err)
			continue
		}
		// go 调度器起的go进程
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	log.Printf("client connected: %s", conn.RemoteAddr())

	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("read failed: %v", err)
			}
			return
		}

		data := buf[:n]
		log.Printf("received: %s", string(data))

		if _, err := conn.Write(data); err != nil {
			log.Printf("write failed: %v", err)
			return
		}
	}
}
