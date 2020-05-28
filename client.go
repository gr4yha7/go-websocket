package main

import (
	"encoding/json"
	"log"
	"time"

	badger "github.com/dgraph-io/badger/v2"
	"github.com/gorilla/websocket"
)

type Client struct {
	id     string
	socket *websocket.Conn
	send   chan []byte
}

type Message struct {
	Sender    string `json:"sender,omitempty"`
	Recipient string `json:"recipient,omitempty"`
	Content   string `json:"content,omitempty"`
}

type SocketInputData struct {
	RollNumber string `json:"roll_number"`
	Message    string `json:"message"`
}

type SocketOutputData struct {
	RollNumber      string `json:"roll_number"`
	TotalWords      int    `json:"total_words"`
	TotalCharacters int    `json:"total_characters"`
}

// add is a func of type MergeFunc which takes in an existing value, and a value to be merged with it
func add(originalValue, newValue []byte) []byte {
	return append(originalValue, newValue...)
}

func addToDB(db *badger.DB, key string, data string) (*badger.MergeOperator, error) {
	keyByte := []byte(key)
	dataByte := append([]byte(" "), []byte(data)...)

	// Badger provides support for ordered merge operations
	// You pass in a func of type MergeFunc which takes in an existing value, and a value to be merged with it
	// It returns a new value which is the result of the merge operation

	m := db.GetMergeOperator(keyByte, add, 200*time.Millisecond)
	defer m.Stop()

	return m, m.Add(dataByte)

}

// Get the values of a key using the merge operator
func getFromDB(m *badger.MergeOperator) (string, error) {
	value, err := m.Get()
	return string(value), err
}

func (c *Client) read() {
	defer func() {
		manager.unregister <- c
		c.socket.Close()
	}()

	var data SocketInputData
	for {
		_, message, err := c.socket.ReadMessage()
		if err != nil {
			manager.unregister <- c
			c.socket.Close()
			break
		}
		if err := json.Unmarshal(message, &data); err != nil {
			log.Panic(err)
		}
		mergeOperator, err = addToDB(db, data.RollNumber, data.Message)
		if err == nil {
			log.Println("Data added to Badger")
		} else {
			log.Fatal(err)
		}

		jsonMessage, _ := json.Marshal(&Message{Sender: c.id, Content: string(message)})
		manager.broadcast <- jsonMessage
	}
}

func (c *Client) write() {
	defer func() {
		c.socket.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.socket.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			var data Message
			if err := json.Unmarshal(message, &data); err != nil {
				log.Panic(err)
			}

			var sockData SocketInputData
			if err := json.Unmarshal([]byte(data.Content), &sockData); err != nil {
				log.Printf("error decoding message content: %v", err)
				if e, ok := err.(*json.SyntaxError); ok {
					log.Printf("syntax error at byte offset %d", e.Offset)
				}
				log.Printf("Message content: %q", data.Content)
				log.Panic(err)
			}

			msg, err := getFromDB(mergeOperator)
			if err != nil {
				return
			}
			totalWords := WordCount(msg)
			totalChars := CharacterCount(msg)

			respData := &SocketOutputData{
				RollNumber:      sockData.RollNumber,
				TotalWords:      totalWords,
				TotalCharacters: totalChars,
			}

			jsonRes, err := json.Marshal(&respData)
			if err != nil {
				log.Panic(err)
			}

			log.Printf("%s sent: Roll No.: %s, Message: %s\n", c.socket.RemoteAddr(), sockData.RollNumber, msg)

			jsonMessage, _ := json.Marshal(&Message{Content: string(jsonRes)})

			c.socket.WriteMessage(websocket.TextMessage, jsonMessage)
		}
	}
} 