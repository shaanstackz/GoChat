package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

type Client struct {
	conn     net.Conn
	send     chan string
	server   *Server
	username string
}

func NewClient(conn net.Conn, server *Server) *Client {
	reader := bufio.NewReader(conn)
	conn.Write([]byte("Enter username: "))
	name, _ := reader.ReadString('\n')

	return &Client{
		conn:     conn,
		send: make(chan string, 20),
		server:   server,
		username: strings.TrimSpace(name),
	}
}

func (c *Client) Read() {
	scanner := bufio.NewScanner(c.conn)

	for scanner.Scan() {
		msg := scanner.Text()

		if strings.HasPrefix(msg, "/sendfile ") {
			filename := strings.TrimSpace(strings.TrimPrefix(msg, "/sendfile "))
			c.send <- "READY"

			reader := bufio.NewReader(c.conn)
			header, _ := reader.ReadString('\n')
			parts := strings.Split(strings.TrimSpace(header), " ")

			size, _ := strconv.Atoi(parts[2])
			buf := make([]byte, size)
			io.ReadFull(c.conn, buf)

			os.WriteFile("uploads/"+filename, buf, 0644)
			c.server.broadcast <- "[SYSTEM] " + c.username + " uploaded " + filename
			continue
		}

		if strings.HasPrefix(msg, "/getfile ") {
			filename := strings.TrimSpace(strings.TrimPrefix(msg, "/getfile "))
			data, err := os.ReadFile("uploads/" + filename)
			if err != nil {
				c.send <- "File not found"
				continue
			}
			header := fmt.Sprintf("FILE %s %d\n", filename, len(data))
			c.conn.Write([]byte(header))
			c.conn.Write(data)
			continue
		}

		c.server.broadcast <- "[" + c.username + "] " + msg
	}

	c.server.unregister <- c
	c.conn.Close()
}

func (c *Client) Write() {
	for msg := range c.send {
		c.conn.Write([]byte(msg + "\n"))
	}
}
