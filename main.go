package main

import (
	"fmt"
	"log"
	"net/http"
)

var bd = "http://www.baidu.com/"

// HelloServer just a demo
func HelloServer(w http.ResponseWriter, req *http.Request) {
	fmt.Println("Inside HelloServer handler")
	fmt.Fprintf(w, "Hello,"+req.URL.Path[1:])
}

func main() {
	http.HandleFunc("/", HelloServer)
	err := http.ListenAndServe("localhost:8765", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}
func checkError(err error) {
	if err != nil {
		log.Fatalf("Get : %v", err)
	}
}
