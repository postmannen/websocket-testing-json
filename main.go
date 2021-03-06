/*
Test loading templates, and use them to be drawn via
a websocket to the browser. The element that is made
in the browser can then be deleted.
The templates are being parsed normally but instead
of executing the template to http.ResponseWriter, we
execute it to a bytes.Buffer which got a io.Writer,
and we then send that buffer over the websocket.
*/
package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"text/template"

	"github.com/gorilla/websocket"
	"github.com/postmannen/mapfile"
)

type message struct {
	Command  string
	Argument string
}

//server will hold all the information needed to run a server,
//and data to be passed around and used by the handlers.
type server struct {
	address string
	//msgToTemplate is a reference to know what html template to
	//be used based on which msg comming in from the client browser.
	msgToTemplateMap map[string]string
	conn             *websocket.Conn
}

func newServer() *server {
	return &server{
		address:          "localhost:8080",
		msgToTemplateMap: make(map[string]string),
	}
}

//socketHandler is the handler who controls all the serverside part
//of the websocket. The other handlers like the rootHandle have to
//load a page containing the JS websocket code to start up the
//communication with the serside websocket.
//This handler is used with all the other handlers if they open a
//websocket on the client side.
func (s *server) socketHandler() http.HandlerFunc {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	var init sync.Once
	var tpl *template.Template
	var err error

	init.Do(func() {
		tpl, err = template.ParseFiles("socketTemplates.gohtml")
		if err != nil {
			log.Printf("error: ParseFiles : %v\n", err)
		}
	})

	return func(w http.ResponseWriter, r *http.Request) {
		//upgrade the handler to a websocket connection
		s.conn, err = upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println("error: websocket Upgrade: ", err)
		}

		//divID is to keep track of the sections sendt to the
		//socket to be shown in the browser.
		divID := 1000

		//msg is to hold the message read from the socket
		//var msg message

		//A channel to know when there is an update on
		//the socket from a client browser
		socketEvent := make(chan message)

		//Read the message from browser.
		//Make a function to check the socket, and if there is an event
		//return the json formatted message, on the socketEvent channel.
		//Refactored this to a Go routine so the whole code doesn't block
		//here waiting for something to arrive on the channel, so we can
		//have multiple input channels.
		go s.checkSocket(socketEvent)

		//Check the different event channels.
		//If more inputs are added to the program, make the program send
		//the input in it's own channel, and handle it here.
		for {
			select {
			case msg := <-socketEvent:
				//In the map that holds all the command to template mappings,
				//check if there is a key in the map that match with
				//the msg comming in on the websocket from browser.
				//If there is no match, whats in msg will be sendt directly back over the socket,
				//to be printed out in the client browser.
				if msg.Command == "executeTemplate" && msg.Argument != "" {
					tplName, ok := s.msgToTemplateMap[msg.Argument]
					if ok {
						//Declare a bytes.Buffer to hold the data for the executed template.
						var tplData bytes.Buffer
						//tplData is a bytes.Buffer, which is a type io.Writer. Here we choose
						//execute the template, but passing the output into tplData insted of
						//'w'. Then we can take the data in tplData and send them over the socket.
						tpl.ExecuteTemplate(&tplData, tplName, divID)
						d := tplData.String()
						//New-lines between the html tags in the template source code
						//is shown in the browser. Trimming awat the new-lines in each line
						//in the template data.
						d = strings.TrimSpace(d)
						msg.Argument = d
					}
					divID++
				}

				//write message back on the socket to the browser
				//err = conn.WriteMessage(msgType, msg)
				err = s.conn.WriteJSON(msg)
				if err != nil {
					fmt.Println("error: WriteMessage failed :", err)
					return
				}
			}

		}
	}
}

//Read the message from browser.
//Make a function to check the socket, and if there is an event
func (s *server) checkSocket(m chan message) {
	for {
		var msg message
		err := s.conn.ReadJSON(&msg)

		if err != nil {
			fmt.Println("error: websocket ReadMessage: ", err)
			return
		}

		//print message to console
		fmt.Printf("Received on server from Client=%v : %v \n", s.conn.RemoteAddr(), msg)
		fmt.Println("Content of msg.Command = ", msg.Command)
		fmt.Println("Content of msg.Argument = ", msg.Argument)

		m <- msg
	}
}

//The rootHandle which is like a normal handle is responsible for
//serving the actual visible root page to the browser, and also
//contains the javascript to be run in the browser.
func (s *server) rootHandle() http.HandlerFunc {
	var init sync.Once
	var tpl *template.Template
	var err error

	return func(w http.ResponseWriter, r *http.Request) {
		init.Do(func() {
			tpl, err = template.ParseFiles("websockets1.html")
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Printf("error: ParseFile : %v\n", err)
			return
		}

		tpl.ExecuteTemplate(w, "websocket", nil)
	}
}

//readMap will use the mapfile package and check for updates in
//the JSON file. If the file is changed convert the JSON read
//to a map of string keys and string values.
//If the file is in the wrong format return the current working
//map.
func (s *server) readMap() {
	updates := make(chan mapfile.Update)
	defer close(updates)

	fw, err := mapfile.New("commandToTemplate.json", updates)
	if err != nil {
		log.Println("Main: Failed to create new FileWatcher struct: ", err)
	}

	err = fw.Watch()
	if err != nil {
		log.Println("Main: Failed to Watch: ", err)
	}

	defer fw.Close()

	for {
		select {
		case u := <-fw.Updates:
			if u.Err == nil {
				fmt.Println("No FileWatch Error: ", u)

				s.msgToTemplateMap, err = fw.Convert(s.msgToTemplateMap)
				if err != nil {
					log.Println("Error: ", err)
				}

				printMap(s.msgToTemplateMap)
			} else {
				fmt.Println("FileWatch Error: ", u)
			}
		}
	}
}

func printMap(m map[string]string) {

	//Print out all the values for testing
	fmt.Println("----------------------------------------------------------------")
	fmt.Println("Content of the map unmarshaled from fileContent :")
	for key, value := range m {
		fmt.Println("key = ", key, "value = ", value)
	}
	fmt.Println("----------------------------------------------------------------")

}

func main() {
	s := newServer()
	//Read JSON file, and create a map of all the msg to template mappings that will occur.
	//The key value is the one to send over a socket to backend.
	go s.readMap()

	fmt.Println("***", s.msgToTemplateMap)
	http.HandleFunc("/echo", s.socketHandler())
	http.HandleFunc("/", s.rootHandle())

	log.Printf("started the web server at %v\n", s.address)
	http.ListenAndServe(s.address, nil)

}
