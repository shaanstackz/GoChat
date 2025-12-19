package server

import (
	"bufio"
	"fmt"
	"net"
)

type Client struct {
	conn   net.Conn
	server *Server
	send   chan string
}

func NewClient(conn net.Conn, server *Server) *Client {
	return &Client{
		conn:   conn,
		server: server,
		send:   make(chan string),
	}
}

func (c *Client) Read() {
	defer func() {
		fmt.Println("Read() exiting")
		c.server.unregister <- c
		c.conn.Close()
	}()

	scanner := bufio.NewScanner(c.conn)
	for scanner.Scan() {
		msg := scanner.Text()
		fmt.Println("Received from client:", msg)
		c.server.broadcast <- msg
	}
}

func (c *Client) Write() {
	writer := bufio.NewWriter(c.conn)
	for msg := range c.send {
		fmt.Fprintln(writer, msg)
		writer.Flush()
	}
}
