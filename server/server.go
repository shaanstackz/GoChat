package server

import (
	"fmt"
	"net"
)

type Server struct {
	clients    map[*Client]bool
	rooms      map[string]map[*Client]bool
	history    map[string][]string
	broadcast  chan string
	register   chan *Client
	unregister chan *Client
}


func NewServer() *Server {
	return &Server{
		clients:    make(map[*Client]bool),
		rooms:      make(map[string]map[*Client]bool),
		history:    make(map[string][]string),
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

		case client := <-s.unregister:
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				close(client.send)
				if oldClients, ok := s.rooms[client.room]; ok {
					delete(oldClients, client)
				}
			}

		case message := <-s.broadcast:
			for client := range s.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(s.clients, client)
					if oldClients, ok := s.rooms[client.room]; ok {
						delete(oldClients, client)
					}
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
		users = users[:len(users)-2]
	}
	return users
}

func (s *Server) ListRooms() string {
	names := ""
	for name := range s.rooms {
		names += name + ", "
	}
	if len(names) > 2 {
		names = names[:len(names)-2]
	}
	return names
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

func (s *Server) MoveClientToRoom(c *Client, room string) {
	if oldClients, ok := s.rooms[c.room]; ok {
		delete(oldClients, c)
	}
	if s.rooms[room] == nil {
		s.rooms[room] = make(map[*Client]bool)
	}
	s.rooms[room][c] = true
}

func (s *Server) BroadcastToRoom(room, msg string) {
	if s.history[room] == nil {
		s.history[room] = []string{}
	}
	s.history[room] = append(s.history[room], msg)
	if len(s.history[room]) > 50 { // keep last 50 messages
		s.history[room] = s.history[room][len(s.history[room])-50:]
	}
	for client := range s.rooms[room] {
		select {
		case client.send <- msg:
		default:
			close(client.send)
			delete(s.clients, client)
			delete(s.rooms[room], client)
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
		go client.Write()
		go client.Read()
	}
}
