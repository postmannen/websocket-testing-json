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
	"sync"
	"text/template"

	"github.com/gorilla/websocket"
)

//server will hold all the information needed to run a server,
//and data to be passed around and used by the handlers.
type server struct {
	address string
}

func newServer() *server {
	return &server{":8080"}
}

//var upgrader = websocket.Upgrader{
//	ReadBufferSize:  1024,
//	WriteBufferSize: 1024,
//}

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
		tpl, err = template.ParseFiles("socketTemplates.html")
		if err != nil {
			log.Printf("error: ParseFiles : %v\n", err)
		}
	})

	return func(w http.ResponseWriter, r *http.Request) {
		//upgrade the handler to a websocket connection
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println("error: websocket Upgrade: ", err)
		}

		//divID is to keep track of the sections sendt to the
		//socket to be shown in the browser.
		divID := 0

		for {
			//read the message from browser
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("error: websocket ReadMessage: ", err)
				return
			}

			//print message to console
			fmt.Printf("Client=%v typed : %v \n", conn.RemoteAddr(), string(msg))

			//Check if the message from browser matches any of the predefined CASE
			//claused below, and execute the the block for that CASE.
			strMsg := string(msg)
			switch strMsg {
			case "addButton":
				//msg = []byte("<button>Test button</button>")
				//Create a buffer to hold all the data in the template.
				//Since bytes.Buffer is a writer we can use it as the
				//destination when executing the template.
				var tplData bytes.Buffer
				tpl.ExecuteTemplate(&tplData, "buttonTemplate1", divID)
				d := tplData.String()
				msg = []byte(d)
				divID++
			case "input":
				msg = []byte("<input placeholder='put something here'></input>")
			case "addTpl":
				//Create a buffer to hold all the data in the template.
				//Since bytes.Buffer is a writer we can use it as the
				//destination when executing the template.
				var tplData bytes.Buffer
				tpl.ExecuteTemplate(&tplData, "socketTemplate1", divID)
				d := tplData.String()
				msg = []byte(d)
				divID++
			case "p":
				//Create a buffer to hold all the data in the template.
				//Since bytes.Buffer is a writer we can use it as the
				//destination when executing the template.
				var tplData bytes.Buffer
				tpl.ExecuteTemplate(&tplData, "paragraphTemplate1", divID)
				d := tplData.String()
				msg = []byte(d)
				divID++
			default:
			}

			//write message back to browser
			err = conn.WriteMessage(msgType, msg)
			if err != nil {
				fmt.Println("error: WriteMessage failed :", err)
				return
			}

		}
	}
}

//The rootHandle which is like a normal handle is responsible for
//serving the actual visible root page to the browser, and also
//contains the javascript to be run in the browser.
func (s *server) rootHandle() http.HandlerFunc {
	var init sync.Once
	var tpl *template.Template
	var err error

	init.Do(func() {
		tpl, err = template.ParseFiles("websockets1.html")
		if err != nil {
			log.Printf("error: ParseFile : %v\n", err)
		}
	})

	return func(w http.ResponseWriter, r *http.Request) {
		tpl.ExecuteTemplate(w, "websocket", nil)
	}
}

func main() {
	s := newServer()
	http.HandleFunc("/echo", s.socketHandler())
	http.HandleFunc("/", s.rootHandle())

	http.ListenAndServe(s.address, nil)

}
