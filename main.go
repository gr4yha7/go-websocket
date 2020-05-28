package main

import (
	"fmt"
	"log"
	"net/http"

	badger "github.com/dgraph-io/badger/v2"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
)

var (
	manager = ClientManager{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
	db            *badger.DB
	mergeOperator *badger.MergeOperator
)

func wsHandler(res http.ResponseWriter, req *http.Request) {
	conn, error := (&websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}).Upgrade(res, req, nil)
	if error != nil {
		http.NotFound(res, req)
		return
	}
	uniqueID, err := uuid.NewV4()
	if err != nil {
		log.Fatal(err)
	}
	client := &Client{id: uniqueID.String(), socket: conn, send: make(chan []byte)}

	manager.register <- client

	go client.read()
	go client.write()
}

func main() {
	var err error
	db, err = badger.Open(badger.DefaultOptions("./tmp/badger"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	fmt.Println("Starting application...")
	go manager.start()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
	http.HandleFunc("/teach", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "teach.html")
	})
	http.HandleFunc("/ws", wsHandler)
	http.ListenAndServe(":12345", nil)
}
