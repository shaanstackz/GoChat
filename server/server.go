package server

import (
	"fmt"
	"net"
)

type Server struct {
	clients    map[*Client]bool
	broadcast  chan string
	register   chan *Client
	unregister chan *Client
}

func NewServer() *Server {
	return &Server{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan string),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (s *Server) Run() {
	for {
		select {
		case client := <-s.register:
			s.clients[client] = true
			fmt.Println("Client connected:", client.username)

		case client := <-s.unregister:
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				close(client.send)
				fmt.Println("Client disconnected:", client.username)
			}

		case message := <-s.broadcast:
			fmt.Println("SERVER broadcasting:", message)
			for client := range s.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(s.clients, client)
				}
			}
		}
	}
}

func (s *Server) ListUsers() string {
	users := ""
	for client := range s.clients {
		users += client.username + ", "
	}
	if len(users) > 2 {
		users = users[:len(users)-2] // remove trailing comma
	}
	return users
}

func (s *Server) SendPrivate(username, msg string) bool {
	for client := range s.clients {
		if client.username == username {
			client.send <- msg
			return true
		}
	}
	return false
}

func Start(addr string) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	fmt.Println("Chat server started on", addr)

	server := NewServer()
	go server.Run()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		client := NewClient(conn, server)
		server.register <- client           
		go client.Write()                   
		go client.Read()                    
	}
}
