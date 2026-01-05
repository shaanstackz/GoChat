package server

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
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
		c.server.RemoveClient(c)
		c.conn.Close()
	}()

	scanner := bufio.NewScanner(c.conn)
	fmt.Fprintln(c.conn, "Enter username:")
	if !scanner.Scan() {
		return
	}
	c.username = scanner.Text()
	c.room = "lobby"
	c.server.AddClient(c, "lobby")

	for scanner.Scan() {
		msg := scanner.Text()

		if msg == "/users" {
			c.send <- c.server.ListUsers()
			continue
		}

		if msg == "/rooms" {
			c.send <- c.server.ListRooms()
			continue
		}

		if strings.HasPrefix(msg, "/join ") {
			room := strings.TrimSpace(strings.TrimPrefix(msg, "/join "))
			c.server.MoveClient(c, room)
			continue
		}

		if strings.HasPrefix(msg, "/msg ") {
			parts := strings.SplitN(msg, " ", 3)
			if len(parts) < 3 {
				continue
			}
			c.server.PrivateMessage(c.username, parts[1], parts[2])
			continue
		}

		if strings.HasPrefix(msg, "/search ") {
			keyword := strings.TrimSpace(strings.TrimPrefix(msg, "/search "))
			results := c.server.Search(c.room, keyword)
			for _, r := range results {
				c.send <- r
			}
			continue
		}

		if strings.HasPrefix(msg, "/edit ") {
			parts := strings.SplitN(msg, " ", 3)
			if len(parts) < 3 {
				continue
			}
			id, _ := strconv.Atoi(parts[1])
			c.server.EditMessage(c, id, parts[2])
			continue
		}

		if strings.HasPrefix(msg, "/delete ") {
			id, _ := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(msg, "/delete ")))
			c.server.DeleteMessage(c, id)
			continue
		}

		c.server.Broadcast(c, msg)
	}
}

func (c *Client) Write() {
	writer := bufio.NewWriter(c.conn)
	for msg := range c.send {
		fmt.Fprintln(writer, msg)
		writer.Flush()
	}
}
