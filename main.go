package main

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"os"
)

func main() {
	go func() {
		ln, err := net.Listen("tcp", ":30000")
		if err != nil {
			log.Fatal(err)
		}
		conn, _ := ln.Accept()
		sendFile(conn, "")
	}()

	// client
}

func sendFile(conn net.Conn, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	defer file.Close()

	const chunkSize = 1024 * 1024 // 1MB
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

		sizeBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(sizeBuf, uint32(n))
		if _, err := conn.Write(sizeBuf); err != nil {
			return err
		}

		if _, err := conn.Write(buf[:n]); err != nil {
			return err
		}
	}

	eofBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(eofBuf, 0)
	_, err = conn.Write(eofBuf)
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
		if _, err := io.ReadFull(conn, sizeBuf); err != nil {
			return err
		}

		size := binary.BigEndian.Uint32(sizeBuf)
		if size == 0 {
			break
		}

		buf := make([]byte, size)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return err
		}

		if _, err := file.Write(buf); err != nil {
			return err
		}
	}
	return nil
}
