package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"golang.org/x/net/websocket"
)

// JSON serializable struct to hold messages
type Message struct{
	Text string `json:"text"`
}

// Holds clients in a map and the channels
type hub struct{
	clients          map[string]*websocket.Conn
	addClientChan    chan *websocket.Conn
	removeClientChan chan *websocket.Conn
	broadcastChan    chan Message   
}

var(
	port = flag.String("port", "9000", "port used for ws connection")
)

func server(port string) error{
	h := newHub()
	mux := http.NewServeMux()
	// Routes / to handler
	mux.Handle("/", websocket.Handler(func(ws *websocket.Conn) {
		handler(ws, h)
	}))

	s := http.Server{Addr: ":" + port, Handler:mux}
	return s.ListenAndServe()
}

// Every client that connects will call this handler
// Runs the hub and then listens for messages from the client
func handler(ws *websocket.Conn, h *hub){
	go h.run()

	h.addClientChan <- ws

	// TODO: Improve error handling
	for {
		var m Message
		err := websocket.JSON.Receive(ws, &m)
		if err != nil {
			h.broadcastChan <- Message{err.Error()}
			h.removeClient(ws)
			return
		}
		h.broadcastChan <- m
	}
}

// Returns a new hub
func newHub() *hub{
	return &hub{
		clients:          make(map[string]*websocket.Conn),
		addClientChan:    make(chan *websocket.Conn),
		removeClientChan: make(chan *websocket.Conn),
		broadcastChan:    make(chan Message),	  
	}
}

// Listens to all the hub's channels
func (h *hub) run(){
	for {
		select {
		case conn := <-h.addClientChan:
			h.addClient(conn)
		case conn := <-h.removeClientChan:
			h.removeClient(conn)
		case m := <-h.broadcastChan:
			h.broadcastMessage(m)
		}
	}
}

func (h *hub) removeClient(conn *websocket.Conn){
	delete(h.clients, conn.LocalAddr().String())
}

func (h *hub) addClient(conn *websocket.Conn) {
	h.clients[conn.RemoteAddr().String()] = conn
}
   
func (h *hub) broadcastMessage(m Message) {
	// Calls JSON.Send for every client in the pool
	for _, conn := range h.clients {
		err := websocket.JSON.Send(conn, m)
		if err != nil {
			fmt.Println("Error broadcasting message: ", err)
			return
		} // if
	} // for
} // broadcastMessage

func main(){
	flag.Parse()
	log.Fatal(server(*port))
}