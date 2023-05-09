package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
            <html>
                <head>
                    <title>Chat</title>
                </head>
                <body>
                    <p>Let's chat!</p>
                </body>
            </html>
        `))
	})
	if err := http.ListenAndServe("localhost:8080", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
