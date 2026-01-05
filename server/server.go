package server

import (
	"fmt"
	"net"
	"strings"
	"time"
)

type Message struct {
	ID      int
	Sender  string
	Content string
	Time    string
}

type Server struct {
	clients map[*Client]bool
	rooms   map[string]map[*Client]bool
	history map[string][]Message
	nextID  int
}

func NewServer() *Server {
	return &Server{
		clients: make(map[*Client]bool),
		rooms:   make(map[string]map[*Client]bool),
		history: make(map[string][]Message),
		nextID:  1,
	}
}

func (s *Server) AddClient(c *Client, room string) {
	s.clients[c] = true
	if s.rooms[room] == nil {
		s.rooms[room] = make(map[*Client]bool)
	}
	s.rooms[room][c] = true
	c.send <- "Joined room " + room
	for _, m := range s.history[room] {
		c.send <- fmt.Sprintf("[%d] [%s] [%s] %s", m.ID, m.Time, m.Sender, m.Content)
	}
	s.BroadcastSystem(room, c.username+" joined")
}

func (s *Server) RemoveClient(c *Client) {
	delete(s.clients, c)
	if s.rooms[c.room] != nil {
		delete(s.rooms[c.room], c)
		s.BroadcastSystem(c.room, c.username+" left")
	}
	close(c.send)
}

func (s *Server) MoveClient(c *Client, room string) {
	if s.rooms[c.room] != nil {
		delete(s.rooms[c.room], c)
	}
	c.room = room
	s.AddClient(c, room)
}

func (s *Server) Broadcast(c *Client, content string) {
	msg := Message{
		ID:      s.nextID,
		Sender:  c.username,
		Content: content,
		Time:    time.Now().Format("15:04"),
	}
	s.nextID++
	s.history[c.room] = append(s.history[c.room], msg)

	formatted := fmt.Sprintf("[%d] [%s] [%s] %s", msg.ID, msg.Time, msg.Sender, msg.Content)
	for client := range s.rooms[c.room] {
		client.send <- formatted
		if strings.Contains(content, "@"+client.username) {
			client.send <- "[MENTION] " + c.username + " mentioned you"
		}
	}
}

func (s *Server) BroadcastSystem(room, msg string) {
	for client := range s.rooms[room] {
		client.send <- "[SYSTEM] " + msg
	}
}

func (s *Server) PrivateMessage(from, to, msg string) {
	for client := range s.clients {
		if client.username == to {
			client.send <- "[PM from " + from + "] " + msg
			return
		}
	}
}

func (s *Server) ListUsers() string {
	var users []string
	for c := range s.clients {
		users = append(users, c.username)
	}
	return "Users: " + strings.Join(users, ", ")
}

func (s *Server) ListRooms() string {
	var rooms []string
	for r := range s.rooms {
		rooms = append(rooms, r)
	}
	return "Rooms: " + strings.Join(rooms, ", ")
}

func (s *Server) Search(room, keyword string) []string {
	var results []string
	for _, m := range s.history[room] {
		if strings.Contains(m.Content, keyword) {
			results = append(results,
				fmt.Sprintf("[%d] [%s] [%s] %s", m.ID, m.Time, m.Sender, m.Content))
		}
	}
	return results
}

func (s *Server) EditMessage(c *Client, id int, text string) {
	for i, m := range s.history[c.room] {
		if m.ID == id && m.Sender == c.username {
			s.history[c.room][i].Content = text
			s.BroadcastSystem(c.room, "message edited")
			return
		}
	}
}

func (s *Server) DeleteMessage(c *Client, id int) {
	for i, m := range s.history[c.room] {
		if m.ID == id && m.Sender == c.username {
			s.history[c.room] = append(s.history[c.room][:i], s.history[c.room][i+1:]...)
			s.BroadcastSystem(c.room, "message deleted")
			return
		}
	}
}

func Start(addr string) {
	ln, _ := net.Listen("tcp", addr)
	fmt.Println("Server on", addr)
	server := NewServer()

	for {
		conn, _ := ln.Accept()
		client := NewClient(conn, server)
		go client.Write()
		go client.Read()
	}
}
