package main

import (
	"fmt"
	"log"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Cosi test web app")
}

func main() {
	http.HandleFunc("/", handler)
	err := http.ListenAndServe(":2379", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
		return
	}
}
