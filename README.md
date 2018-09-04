# websocket test using JSON

The idea is:

* The user do some action in the browser. For example ask the server for a Go Web Template to be drawn.
* The user action is a JSON object in the form {"Command":"xxxx","Argument":"yyyy"} where Command can be "executeTemplate", and Action can be "someTemplate".
* Send the JSON object over the Socket to the Server.
* The Server then Decodes the JSON object.
* Then the server checks the Command, and if it is for example "executeTemplate" then it will check it's map containing all the Argument to Template mappings, and pick the correct template.
* The template is then Executed on the server, but instead of writing the template to http.ResponseWriter we write the template to a bytes.Buffer. The buffer is then stripped of leading and ending spaces, and converted to a string, and put into the JSON object.
* The Json object containing the template (pure html code) is then put onto the socket and sendt to the client.
* The client reads the socket and renders the html on the page in the browser.
* The JSON of commands is read from file, and then converted to a map.
* If there is a change in the JSON file the map is automatically updated when file is changed. This is done with the mapfile package.

![alt text](https://github.com/postmannen/websocket-testing-json/blob/master/doc/websocket-diagram.png)

## Structure of JSON

* {"Command":"executeTemplate","Argument":"addHeader"}
* {"Command":"executeTemplate","Argument":"addButton"}

## Structure of Argument to Template map

    s.msgToTemplateMap = map[string]string{
        "addButton":    "buttonTemplate1",
        "addHeader":    "socketTemplate1",
        "addParagraph": "paragraphTemplate1",
    }
