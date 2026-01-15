package main

import (
	"fmt"
	"net"
	"os"
)

type Message struct {
	room string
	text string
}

type Server struct {
	clients    map[*Client]bool
	rooms      map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan Message
}

func NewServer() *Server {
	return &Server{
		clients:    make(map[*Client]bool),
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client, 10),
		unregister: make(chan *Client, 10),
		broadcast:  make(chan Message, 100),
	}
}

func (s *Server) Run() {
	for {
		select {

		case c := <-s.register:
			s.clients[c] = true
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
			delete(s.rooms[c.room], c)
			close(c.send)

		case msg := <-s.broadcast:
			for c := range s.rooms[msg.room] {
				select {
				case c.send <- msg.text:
				default:
				}
			}
		}
	}
}

func main() {
	os.MkdirAll("uploads", 0755)

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
