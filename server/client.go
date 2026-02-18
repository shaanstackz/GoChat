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
				continue
			}
			if c.server.UsernameExists(newName) {
				c.send <- "Username already taken"
				continue
			}

			delete(c.server.users, c.username)
			c.username = newName
			c.server.users[newName] = c
			continue
		}

		if strings.HasPrefix(msg, "/dm ") {
			parts := strings.SplitN(msg, " ", 3)
			if len(parts) < 3 {
				continue
			}
			c.server.PrivateMessage(c.username, parts[1], parts[2])
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

			c.send <- "Joined " + newRoom
			c.server.sendRoomHistory(c)
			continue
		}

		c.server.broadcast <- Message{
			room: c.room,
			user: c.username,
			text: msg,
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
