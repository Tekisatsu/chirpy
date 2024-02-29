package main

import (
	"log"
	"net/http"
)
	
func NewServeMux() *http.ServeMux {
	return http.NewServeMux()
}
func middlewareCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func main () {
	mux := NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
		})
	corsMux := middlewareCors(mux)
	srv := &http.Server {
		Addr: "localhost:8080",
		Handler: corsMux,
	}
	log.Fatal(srv.ListenAndServe())
}
