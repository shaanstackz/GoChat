package main

import (
	"bufio"
	"net"
	"strings"
)

type Client struct {
	conn     net.Conn
	send     chan string
	server   *Server
	username string
	room     string
}

func NewClient(conn net.Conn, server *Server) *Client {
	reader := bufio.NewReader(conn)
	conn.Write([]byte("Enter username: "))
	name, _ := reader.ReadString('\n')

	return &Client{
		conn:     conn,
		send:     make(chan string, 20),
		server:   server,
		username: strings.TrimSpace(name),
		room:     "general",
	}
}

func (c *Client) Read() {
	scanner := bufio.NewScanner(c.conn)

	for scanner.Scan() {
		msg := scanner.Text()

		if strings.HasPrefix(msg, "/nick ") {
			newName := strings.TrimSpace(strings.TrimPrefix(msg, "/nick "))
			if newName == "" {
				c.send <- "Usage: /nick <newname>"
				continue
			}
			if c.server.UsernameExists(newName) {
				c.send <- "Username already taken"
				continue
			}

			old := c.username
			delete(c.server.users, old)
			c.username = newName
			c.server.users[newName] = c

			c.server.broadcast <- Message{
				room: c.room,
				text: "[SYSTEM] " + old + " is now known as " + newName,
			}
			continue
		}

		if strings.HasPrefix(msg, "/dm ") {
			parts := strings.SplitN(msg, " ", 3)
			if len(parts) < 3 {
				c.send <- "Usage: /dm <user> <message>"
				continue
			}
			c.server.PrivateMessage(c.username, parts[1], parts[2])
			continue
		}

		if msg == "/history" {
			for _, m := range c.server.roomHistory[c.room] {
				c.send <- m
			}
			continue
		}

		if strings.HasPrefix(msg, "/history dm ") {
			user := strings.TrimSpace(strings.TrimPrefix(msg, "/history dm "))
			h := c.server.dmHistory[c.username][user]
			if len(h) == 0 {
				c.send <- "No DM history"
				continue
			}
			for _, m := range h {
				c.send <- m
			}
			continue
		}

		if strings.HasPrefix(msg, "/join ") {
			newRoom := strings.TrimSpace(strings.TrimPrefix(msg, "/join "))
			delete(c.server.rooms[c.room], c)
			c.room = newRoom

			if c.server.rooms[newRoom] == nil {
				c.server.rooms[newRoom] = make(map[*Client]bool)
			}
			c.server.rooms[newRoom][c] = true

			c.send <- "Joined room " + newRoom
			for _, m := range c.server.roomHistory[newRoom] {
				c.send <- m
			}
			continue
		}

		if msg == "/rooms" {
			for r := range c.server.rooms {
				c.send <- r
			}
			continue
		}

		if msg == "/who" {
			for u := range c.server.rooms[c.room] {
				c.send <- u.username
			}
			continue
		}

		c.server.broadcast <- Message{
			room: c.room,
			text: "[" + c.username + "] " + msg,
		}
	}

	c.server.unregister <- c
	c.conn.Close()
}

func (c *Client) Write() {
	for msg := range c.send {
		c.conn.Write([]byte(msg + "\n"))
	}
}
