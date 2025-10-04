package main

import (
	"net/http"
	"net/url"
	"strings"
)

type MemStorage struct {
	GaugeMetrics   map[string]float64
	CounterMetrics map[string]int64
}

func updateHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodPost {
		u, err := url.Parse(r.URL.Path)
		if err != nil {
			http.Error(w, "Wrong url", http.StatusBadRequest)
		}
		parts := strings.Split(u.Path, "/")[2:]
		if len(parts) != 3 {
			http.Error(w, "Wrong metrics data", http.StatusBadRequest)
			return
		}

		if strings.ToLower(parts[0]) == "gauge" {

		} else if strings.ToLower(parts[0]) == "counter" {

		} else {
			http.Error(w, "Wrong metrics type", http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write(nil)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/update/", updateHandler)

	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}

}
