package server

import (
	"fmt"
	"net"
)

type Server struct {
	clients   map[*Client]bool
	broadcast chan string
	register  chan *Client
	unregister chan *Client
}

func NewServer() *Server {
	return &Server{
		clients:   make(map[*Client]bool),
		broadcast: make(chan string),
		register:  make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (s *Server) Run() {
	for {
		select {
		case client := <-s.register:
			s.clients[client] = true
			fmt.Println("Client connected")

		case client := <-s.unregister:
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				close(client.send)
				fmt.Println("Client disconnected")
			}

		case message := <-s.broadcast:
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
		go client.Read()                    
		go client.Write()                   
	}
}

