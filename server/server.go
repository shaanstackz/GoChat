package main

import (
	"database/sql"
	"fmt"
	"net"

	_ "github.com/mattn/go-sqlite3"
)

type Message struct {
	room string
	user string
	text string
}

type Server struct {
	db         *sql.DB
	clients    map[*Client]bool
	users      map[string]*Client
	rooms      map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan Message
}

func NewServer() *Server {
	db, err := sql.Open("sqlite3", "./chat.db")
	if err != nil {
		panic(err)
	}

	initDB(db)

	return &Server{
		db:         db,
		clients:    make(map[*Client]bool),
		users:      make(map[string]*Client),
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client, 10),
		unregister: make(chan *Client, 10),
		broadcast:  make(chan Message, 100),
	}
}

func initDB(db *sql.DB) {
	db.Exec(`
	CREATE TABLE IF NOT EXISTS room_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		room TEXT,
		user TEXT,
		message TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	db.Exec(`
	CREATE TABLE IF NOT EXISTS dm_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		from_user TEXT,
		to_user TEXT,
		message TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
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

			s.sendRoomHistory(c)

		case c := <-s.unregister:
			delete(s.clients, c)
			delete(s.users, c.username)
			delete(s.rooms[c.room], c)
			close(c.send)

		case msg := <-s.broadcast:

			s.saveRoomMessage(msg.room, msg.user, msg.text)

			formatted := "[" + msg.user + "] " + msg.text

			for c := range s.rooms[msg.room] {
				select {
				case c.send <- formatted:
				default:
				}
			}
		}
	}
}

func (s *Server) saveRoomMessage(room, user, text string) {
	s.db.Exec(
		"INSERT INTO room_messages(room, user, message) VALUES(?,?,?)",
		room, user, text,
	)
}

func (s *Server) saveDM(from, to, msg string) {
	s.db.Exec(
		"INSERT INTO dm_messages(from_user, to_user, message) VALUES(?,?,?)",
		from, to, msg,
	)
}

func (s *Server) sendRoomHistory(c *Client) {
	rows, _ := s.db.Query(`
		SELECT user, message
		FROM room_messages
		WHERE room=?
		ORDER BY id DESC
		LIMIT 20
	`, c.room)

	defer rows.Close()

	var msgs []string

	for rows.Next() {
		var u, m string
		rows.Scan(&u, &m)
		msgs = append(msgs, "["+u+"] "+m)
	}

	for i := len(msgs) - 1; i >= 0; i-- {
		c.send <- msgs[i]
	}
}

func (s *Server) PrivateMessage(from, to, msg string) {
	s.saveDM(from, to, msg)

	target, ok := s.users[to]
	if !ok {
		s.users[from].send <- "User not found"
		return
	}

	target.send <- "[DM from " + from + "] " + msg
	s.users[from].send <- "[DM to " + to + "] " + msg
}

func (s *Server) UsernameExists(name string) bool {
	_, ok := s.users[name]
	return ok
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
