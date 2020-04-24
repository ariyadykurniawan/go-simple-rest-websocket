package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Message struct {
	Content string `json:"message"`
}

var Messages []string

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WebSocketConnection struct {
	*websocket.Conn
}

var connections = make([]*WebSocketConnection, 0)

func main() {
	fmt.Println("Start warpin message app")
	route()
}

func route() {
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/messages", returnAllMessages).Methods("GET")
	myRouter.HandleFunc("/message", createNewMessage).Methods("POST")
	myRouter.HandleFunc("/message/realtime", wsEndpoint)
	log.Fatal(http.ListenAndServe(":8080", myRouter))
}

func returnAllMessages(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint Hit: returnAllMessages")
	w.Header().Set("Content-Type", "application/json")
	if len(Messages) == 0 {
		fmt.Fprintf(w, "No message exist")
	} else {
		json.NewEncoder(w).Encode(Messages)
	}

}

func createNewMessage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint Hit: createNewMessage")
	reqBody, _ := ioutil.ReadAll(r.Body)
	var message Message
	json.Unmarshal(reqBody, &message)
	Messages = append(Messages, message.Content)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(message)
}

func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
	}

	log.Println("Client Connected")
	err = ws.WriteMessage(1, []byte("Hi Client!"))
	if err != nil {
		log.Println(err)
	}

	currentConn := WebSocketConnection{Conn: ws}
	connections = append(connections, &currentConn)

	go handleIO(&currentConn, connections)
}

func handleIO(currentConn *WebSocketConnection, connections []*WebSocketConnection) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("ERROR", fmt.Sprintf("%v", r))
		}
	}()
	for {
		messageType, p, err := currentConn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		fmt.Println(string(p))

		broadcastMessage(currentConn, messageType, p)

	}
}

func broadcastMessage(currentConn *WebSocketConnection, messageType int, p []byte) {
	for _, eachConn := range connections {
		eachConn.WriteMessage(messageType, p)
	}
	Messages = append(Messages, string(p))
}
