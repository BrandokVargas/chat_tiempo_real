package main

import (
	"log"
	"net/http"
)

func main() {

	hub := newHub()

	go hub.run()

	serveMux := http.NewServeMux()

	serveMux.Handle("/", http.FileServer(http.Dir("public")))

	serveMux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		hanldeWS(hub, w, r)
	})

	log.Println("running server on port :8080")
	log.Println(http.ListenAndServe(":8080", serveMux))
}
