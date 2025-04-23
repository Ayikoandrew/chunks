package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage:")
		fmt.Println("  As server: go run practice.go server [port]")
		fmt.Println("  As client: go run practice.go client [server_address:port] [filename]")
		return
	}

	mode := os.Args[1]

	switch mode {
	case "server":
		if len(os.Args) < 3 {
			fmt.Println("Missing port number")
			return
		}
		port := os.Args[2]
		runServer(":" + port)
	case "client":
		if len(os.Args) < 4 {
			fmt.Println("Missing server address or filename")
			return
		}
		serverAddr := os.Args[2]
		filename := os.Args[3]
		runClient(serverAddr, filename)
	default:
		fmt.Println("Unknown mode. Use 'server' or 'client'")
	}
}

func runServer(address string) error {
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	originalFilename, err := receiveFilename(conn)
	if err != nil {
		fmt.Println("Error receiving filename:", err)
		return
	}

	safeFilename := filepath.Base(originalFilename)

	finalFilename := safeFilename
	for i := 1; fileExists(finalFilename); i++ {
		ext := filepath.Ext(safeFilename)
		basename := safeFilename[:len(safeFilename)-len(ext)]
		finalFilename = fmt.Sprintf("%s_%d%s", basename, i, ext)
	}

	err = receiveFile(conn, finalFilename)
	if err != nil {
		fmt.Println("Error receiving file:", err)
		os.Remove(finalFilename)
		return
	}
	fmt.Println("File received and saved as:", finalFilename)
}

func fileExists(finalFilename string) bool {
	_, err := os.Stat(finalFilename)
	return os.IsNotExist(err)
}

func receiveFilename(conn net.Conn) (string, error) {
	sizeBuf := make([]byte, 4)

	_, err := io.ReadFull(conn, sizeBuf)
	if err != nil {
		return "", err
	}

	size := binary.BigEndian.Uint32(sizeBuf)

	buf := make([]byte, size)
	_, err = io.ReadFull(conn, buf)
	if err != nil {
		return "", err
	}

	return string(buf), nil
}

func runClient(serverAddr, filename string) {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer conn.Close()
	fmt.Println("Connected to server, sending file:", filename)

	err = sendFilename(conn, filename)
	if err != nil {
		fmt.Println("Error sending filename:", err)
		return
	}

	err = sendFile(conn, filename)
	if err != nil {
		fmt.Println("Error sending file:", err)
		return
	}
	fmt.Println("File sent successfully")
}

func sendFilename(conn net.Conn, filename string) error {
	name := filepath.Base(filename)

	nameBytes := []byte(name)
	sizeBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(sizeBuf, uint32(len(nameBytes)))
	_, err := conn.Write(sizeBuf)
	if err != nil {
		return err
	}

	_, err = conn.Write(nameBytes)
	return err
}

func sendFile(conn net.Conn, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var chunkSize = 1024 * 1024 // 1MB
	buf := make([]byte, chunkSize)
	for {
		n, err := file.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if n == 0 {
			break
		}

		datasize := uint32(n)

		sizeBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(sizeBuf, datasize)

		_, err = conn.Write(sizeBuf)

		if err != nil {
			return err
		}

		_, err = conn.Write(buf[:n])
		if err != nil {
			return err
		}

	}

	eofSize := make([]byte, 4)
	binary.BigEndian.PutUint32(eofSize, 0)
	_, err = conn.Write(eofSize)
	return err
}

func receiveFile(conn net.Conn, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer file.Close()

	sizeBuf := make([]byte, 4)

	for {
		_, err := io.ReadFull(conn, sizeBuf)
		if err != nil {
			return err
		}

		size := binary.BigEndian.Uint32(sizeBuf)

		if size == 0 {
			break
		}

		buf := make([]byte, size)
		_, err = io.ReadFull(conn, buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		_, err = file.Write(buf)
		if err != nil {
			return err
		}
	}
	return nil
}
