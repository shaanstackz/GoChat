package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

type Client struct {
	conn     net.Conn
	server   *Server
	send     chan string
	username string
}

func NewClient(conn net.Conn, server *Server) *Client {
	return &Client{
		conn:   conn,
		server: server,
		send:   make(chan string, 10), 
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

		if msg == "/users" {
			userList := c.server.ListUsers()
			c.send <- "Connected users: " + userList
			continue
		}

		if strings.HasPrefix(msg, "/msg ") {
			parts := strings.SplitN(msg, " ", 3)
			if len(parts) < 3 {
				c.send <- "Usage: /msg username message"
				continue
			}
			targetName := parts[1]
			privateMsg := parts[2]
			if !c.server.SendPrivate(targetName, fmt.Sprintf("[PM from %s] %s", c.username, privateMsg)) {
				c.send <- "User not found: " + targetName
			}
			continue
		}

		if strings.HasPrefix(msg, "/nick ") {
			newName := strings.TrimSpace(strings.TrimPrefix(msg, "/nick "))
			if newName == "" {
				c.send <- "Usage: /nick newname"
				continue
			}
			oldName := c.username
			c.username = newName
			c.server.broadcast <- fmt.Sprintf("🔄 %s changed name to %s", oldName, newName)
			continue
		}

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
