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
	room     string
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
		c.server.BroadcastToRoom(c.room, fmt.Sprintf("🔴 %s left the room", c.username))
		c.server.unregister <- c
		c.conn.Close()
	}()

	scanner := bufio.NewScanner(c.conn)
	fmt.Fprintln(c.conn, "Enter username:")
	if !scanner.Scan() {
		return
	}
	c.username = scanner.Text()
	c.room = "lobby"
	c.server.MoveClientToRoom(c, "lobby")
	if messages, ok := c.server.history["lobby"]; ok {
		for _, msg := range messages {
			c.send <- msg
		}
	}
	c.send <- "You joined room lobby"
	c.server.BroadcastToRoom("lobby", fmt.Sprintf("🔵 %s joined the room", c.username))

	for scanner.Scan() {
		msg := scanner.Text()

		if msg == "/users" {
			c.send <- "Connected users: " + c.server.ListUsers()
			continue
		}

		if msg == "/rooms" {
			c.send <- "Active rooms: " + c.server.ListRooms()
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
			c.server.BroadcastToRoom(c.room, fmt.Sprintf("🔄 %s changed name to %s", oldName, newName))
			continue
		}

		if strings.HasPrefix(msg, "/join ") {
			newRoom := strings.TrimSpace(strings.TrimPrefix(msg, "/join "))
			if newRoom == "" {
				c.send <- "Usage: /join roomname"
				continue
			}
			oldRoom := c.room
			c.server.MoveClientToRoom(c, newRoom)
			c.room = newRoom
			if messages, ok := c.server.history[newRoom]; ok {
				for _, m := range messages {
					c.send <- m
				}
			}
			c.send <- fmt.Sprintf("You joined room %s", newRoom)
			c.server.BroadcastToRoom(oldRoom, fmt.Sprintf("🔴 %s left the room", c.username))
			c.server.BroadcastToRoom(newRoom, fmt.Sprintf("🔵 %s joined the room", c.username))
			continue
		}

		formatted := fmt.Sprintf("[%s] %s", c.username, msg)
		c.server.BroadcastToRoom(c.room, formatted)
	}
}

func (c *Client) Write() {
	writer := bufio.NewWriter(c.conn)
	for msg := range c.send {
		fmt.Fprintln(writer, msg)
		writer.Flush()
	}
}
