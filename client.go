// package main

// import (
// 	"encoding/json"
// 	"fmt"
// 	"log"
// 	"net/http"
// 	"regexp"
// 	"strings"
// 	"time"
// 	"unicode"

// 	badger "github.com/dgraph-io/badger/v2"
// 	"github.com/gofrs/uuid"
// 	"github.com/gorilla/websocket"
// )

// type ClientManager struct {
// 	clients    map[*Client]bool
// 	broadcast  chan []byte
// 	register   chan *Client
// 	unregister chan *Client
// }

// type Client struct {
// 	id     string
// 	socket *websocket.Conn
// 	send   chan []byte
// }

// type SocketInputData struct {
// 	RollNumber string `json:"roll_number"`
// 	Message    string `json:"message"`
// }

// type SocketOutputData struct {
// 	RollNumber      string `json:"roll_number"`
// 	TotalWords      int    `json:"total_words"`
// 	TotalCharacters int    `json:"total_characters"`
// }

// type Message struct {
// 	Sender    string `json:"sender,omitempty"`
// 	Recipient string `json:"recipient,omitempty"`
// 	Content   string `json:"content,omitempty"`
// }

// var (
// 	manager = ClientManager{
// 		broadcast:  make(chan []byte),
// 		register:   make(chan *Client),
// 		unregister: make(chan *Client),
// 		clients:    make(map[*Client]bool),
// 	}
// 	db            *badger.DB
// 	mergeOperator *badger.MergeOperator
// )

// func WordCount(value string) int {
// 	// Match non-space character sequences.
// 	re := regexp.MustCompile(`[\S]+`)

// 	// Find all matches and return count.
// 	results := re.FindAllString(value, -1)
// 	return len(results)
// }

// func removeWhitespace(str string) string {
// 	return strings.Map(func(r rune) rune {
// 		if unicode.IsSpace(r) {
// 			return -1
// 		}
// 		return r
// 	}, str)
// }

// func CharacterCount(word string) int {
// 	// remove whitespaces
// 	wordNoWhitespace := removeWhitespace(word)
// 	return len([]rune(wordNoWhitespace))
// }

// // add is a func of type MergeFunc which takes in an existing value, and a value to be merged with it
// func add(originalValue, newValue []byte) []byte {
// 	return append(originalValue, newValue...)
// }

// func addToDB(db *badger.DB, key string, data string) (*badger.MergeOperator, error) {
// 	keyByte := []byte(key)
// 	dataByte := []byte(data)

// 	// Badger provides support for ordered merge operations
// 	// You pass in a func of type MergeFunc which takes in an existing value, and a value to be merged with it
// 	// It returns a new value which is the result of the merge operation

// 	m := db.GetMergeOperator(keyByte, add, 200*time.Millisecond)
// 	defer m.Stop()

// 	return m, m.Add(dataByte)

// }

// func getFromDB(m *badger.MergeOperator) (string, error) {
// 	value, err := m.Get()
// 	return string(value), err
// }

// func (manager *ClientManager) start() {
// 	for {
// 		select {
// 		case conn := <-manager.register:
// 			manager.clients[conn] = true
// 			// jsonMessage, _ := json.Marshal(&Message{Content: "A new socket has connected."})
// 			// manager.send(jsonMessage, conn)
// 		case conn := <-manager.unregister:
// 			if _, ok := manager.clients[conn]; ok {
// 				close(conn.send)
// 				delete(manager.clients, conn)
// 				// jsonMessage, _ := json.Marshal(&Message{Content: "A socket has disconnected."})
// 				// manager.send(jsonMessage, conn)
// 			}
// 		case message := <-manager.broadcast:
// 			for conn := range manager.clients {
// 				select {
// 				case conn.send <- message:
// 				default:
// 					close(conn.send)
// 					delete(manager.clients, conn)
// 				}
// 			}
// 		}
// 	}
// }

// func (manager *ClientManager) send(message []byte, ignore *Client) {
// 	for conn := range manager.clients {
// 		if conn != ignore {
// 			conn.send <- message
// 		}
// 	}
// }

// func (c *Client) read() {
// 	defer func() {
// 		manager.unregister <- c
// 		c.socket.Close()
// 	}()

// 	var data SocketInputData
// 	for {
// 		_, message, err := c.socket.ReadMessage()
// 		if err != nil {
// 			manager.unregister <- c
// 			c.socket.Close()
// 			break
// 		}
// 		if err := json.Unmarshal(message, &data); err != nil {
// 			log.Panic(err)
// 		}
// 		mergeOperator, err = addToDB(db, data.RollNumber, data.Message)
// 		if err == nil {
// 			log.Println("Data added to Badger")
// 		} else {
// 			log.Fatal(err)
// 		}

// 		jsonMessage, _ := json.Marshal(&Message{Sender: c.id, Content: string(message)})
// 		manager.broadcast <- jsonMessage
// 	}
// }

// func (c *Client) write() {
// 	defer func() {
// 		c.socket.Close()
// 	}()

// 	for {
// 		select {
// 		case message, ok := <-c.send:
// 			if !ok {
// 				c.socket.WriteMessage(websocket.CloseMessage, []byte{})
// 				return
// 			}

// 			var data Message
// 			if err := json.Unmarshal(message, &data); err != nil {
// 				log.Panic(err)
// 			}

// 			var sockData SocketInputData
// 			if err := json.Unmarshal([]byte(data.Content), &sockData); err != nil {
// 				log.Printf("error decoding message content: %v", err)
// 				if e, ok := err.(*json.SyntaxError); ok {
// 					log.Printf("syntax error at byte offset %d", e.Offset)
// 				}
// 				log.Printf("Message content: %q", data.Content)
// 				log.Panic(err)
// 			}

// 			msg, err := getFromDB(mergeOperator)
// 			if err != nil {
// 				return
// 			}
// 			totalWords := WordCount(msg)
// 			totalChars := CharacterCount(msg)

// 			respData := &SocketOutputData{
// 				RollNumber:      sockData.RollNumber,
// 				TotalWords:      totalWords,
// 				TotalCharacters: totalChars,
// 			}

// 			jsonRes, err := json.Marshal(&respData)
// 			if err != nil {
// 				log.Panic(err)
// 			}

// 			jsonMessage, _ := json.Marshal(&Message{Content: string(jsonRes)})

// 			c.socket.WriteMessage(websocket.TextMessage, jsonMessage)
// 		}
// 	}
// }

// func wsHandler(res http.ResponseWriter, req *http.Request) {
// 	conn, error := (&websocket.Upgrader{
// 		CheckOrigin: func(r *http.Request) bool {
// 			return true
// 		},
// 	}).Upgrade(res, req, nil)
// 	if error != nil {
// 		http.NotFound(res, req)
// 		return
// 	}
// 	uniqueID, err := uuid.NewV4()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	client := &Client{id: uniqueID.String(), socket: conn, send: make(chan []byte)}

// 	manager.register <- client

// 	go client.read()
// 	go client.write()
// }

// func main() {
// 	var err error
// 	db, err = badger.Open(badger.DefaultOptions("./tmp/badger"))
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer db.Close()
// 	fmt.Println("Starting application...")
// 	go manager.start()
// 	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
// 		http.ServeFile(w, r, "index.html")
// 	})
// 	// http.HandleFunc("/teach", func(w http.ResponseWriter, r *http.Request) {
// 	// 	http.ServeFile(w, r, "teach.html")
// 	// })
// 	http.HandleFunc("/ws", wsHandler)
// 	http.ListenAndServe(":12345", nil)
// }
