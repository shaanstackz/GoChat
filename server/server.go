package main

import (
	"fmt"
	"net"
)

type Message struct {
	room string
	text string
}

type Server struct {
	clients     map[*Client]bool
	users       map[string]*Client
	rooms       map[string]map[*Client]bool
	roomHistory map[string][]string
	dmHistory   map[string]map[string][]string
	register    chan *Client
	unregister  chan *Client
	broadcast   chan Message
}

func NewServer() *Server {
	return &Server{
		clients:     make(map[*Client]bool),
		users:       make(map[string]*Client),
		rooms:       make(map[string]map[*Client]bool),
		roomHistory: make(map[string][]string),
		dmHistory:   make(map[string]map[string][]string),
		register:    make(chan *Client, 10),
		unregister:  make(chan *Client, 10),
		broadcast:   make(chan Message, 100),
	}
}

func (s *Server) Run() {
	for {
		select {

		case c := <-s.register:
			s.clients[c] = true
			s.users[c.username] = c

			if s.rooms[c.room] == nil {
				s.rooms[c.room] = make(map[*Client]bool)
			}
			s.rooms[c.room][c] = true

			s.broadcast <- Message{
				room: c.room,
				text: "[SYSTEM] " + c.username + " joined " + c.room,
			}

		case c := <-s.unregister:
			delete(s.clients, c)
			delete(s.users, c.username)
			delete(s.rooms[c.room], c)
			close(c.send)

		case msg := <-s.broadcast:

			h := s.roomHistory[msg.room]
			h = append(h, msg.text)
			if len(h) > 50 {
				h = h[len(h)-50:]
			}
			s.roomHistory[msg.room] = h

			for c := range s.rooms[msg.room] {
				select {
				case c.send <- msg.text:
				default:
				}
			}
		}
	}
}

func (s *Server) UsernameExists(name string) bool {
	_, ok := s.users[name]
	return ok
}

func (s *Server) PrivateMessage(from, to, msg string) {
	if s.dmHistory[from] == nil {
		s.dmHistory[from] = make(map[string][]string)
	}
	if s.dmHistory[to] == nil {
		s.dmHistory[to] = make(map[string][]string)
	}

	formatted := "[DM " + from + "→" + to + "] " + msg

	s.dmHistory[from][to] = append(s.dmHistory[from][to], formatted)
	s.dmHistory[to][from] = append(s.dmHistory[to][from], formatted)

	if len(s.dmHistory[from][to]) > 30 {
		s.dmHistory[from][to] = s.dmHistory[from][to][1:]
	}
	if len(s.dmHistory[to][from]) > 30 {
		s.dmHistory[to][from] = s.dmHistory[to][from][1:]
	}

	target, ok := s.users[to]
	if !ok {
		s.users[from].send <- "User not found"
		return
	}

	target.send <- "[DM from " + from + "] " + msg
	s.users[from].send <- "[DM to " + to + "] " + msg
}

func main() {
	server := NewServer()
	go server.Run()

	ln, _ := net.Listen("tcp", ":9000")
	fmt.Println("Chat server started on :9000")

	for {
		conn, _ := ln.Accept()
		client := NewClient(conn, server)
		server.register <- client
		go client.Write()
		go client.Read()
	}
}
