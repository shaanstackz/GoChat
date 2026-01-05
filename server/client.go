package server

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"os"
	"io"
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

		if strings.HasPrefix(msg, "FILE ") {
			parts := strings.Split(msg, " ")
			filename := parts[1]
			size, _ := strconv.Atoi(parts[2])

			buf := make([]byte, size)
			io.ReadFull(c.conn, buf)

			os.WriteFile("uploads/"+filename, buf, 0644)

			c.server.BroadcastSystem(
				c.room,
				fmt.Sprintf("%s uploaded file %s (%d bytes)", c.username, filename, size),
			)
			continue
		}


		if strings.HasPrefix(msg, "/sendfile ") {
			filename := strings.TrimSpace(strings.TrimPrefix(msg, "/sendfile "))
			data, err := os.ReadFile(filename)
			if err != nil {
				c.send <- "File not found"
				continue
			}

			header := fmt.Sprintf("FILE %s %d\n", filename, len(data))
			c.conn.Write([]byte(header))
			c.conn.Write(data)

			c.send <- "File sent: " + filename
			continue
		}

		if strings.HasPrefix(msg, "/getfile ") {
			filename := strings.TrimSpace(strings.TrimPrefix(msg, "/getfile "))
			path := "uploads/" + filename

			data, err := os.ReadFile(path)
			if err != nil {
				c.send <- "File not found"
				continue
			}

			header := fmt.Sprintf("FILE %s %d\n", filename, len(data))
			c.conn.Write([]byte(header))
			c.conn.Write(data)
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
