package server

import (
	"bufio"
	"fmt"
	"net"
)

type Client struct {
	conn     net.Conn
	server   *Server
	send     chan string
	username string
}


func NewClient(conn net.Conn, server *Server) *Client {
	return &Client{
		conn:     conn,
		server:   server,
		send: make(chan string, 10),
	}
}


func (c *Client) Read() {
	defer func() {
		c.server.broadcast <- fmt.Sprintf("🔴 %s left the chat", c.username)
		c.server.unregister <- c
		c.conn.Close()
	}()

	scanner := bufio.NewScanner(c.conn)

	fmt.Fprintln(c.conn, "Enter username:")
	if !scanner.Scan() {
		return
	}
	c.username = scanner.Text()  

	c.server.broadcast <- fmt.Sprintf("🔵 %s joined the chat", c.username)

	for scanner.Scan() {
		msg := scanner.Text()
		formatted := fmt.Sprintf("[%s] %s", c.username, msg)
		fmt.Println("READ:", formatted)   
		c.server.broadcast <- formatted
	}
}


func (c *Client) Write() {
	writer := bufio.NewWriter(c.conn)
	for msg := range c.send {
		fmt.Fprintln(writer, msg)
		writer.Flush()
	}
}
