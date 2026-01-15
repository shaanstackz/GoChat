package main

import (
	"fmt"
	"net"
	"os"
)

type Server struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan string
}

func NewServer() *Server {
	return &Server{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client, 10),
		unregister: make(chan *Client, 10),
		broadcast:  make(chan string, 100),
	}
}

func (s *Server) Run() {
	for {
		select {
		case c := <-s.register:
			s.clients[c] = true
			s.broadcast <- "[SYSTEM] " + c.username + " joined"
		case c := <-s.unregister:
			delete(s.clients, c)
			close(c.send)
			s.broadcast <- "[SYSTEM] " + c.username + " left"
		case msg := <-s.broadcast:
			for c := range s.clients {
				c.send <- msg
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
		go client.Read()
		go client.Write()
	}
}
