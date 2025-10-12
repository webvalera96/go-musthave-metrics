package main

import (
	"github.com/webvalera96/go-musthave-metrics/internal/handler"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/update/", handler.Update)

	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}

}
