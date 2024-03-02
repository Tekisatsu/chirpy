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
	mux.Handle("/app/",http.StripPrefix("/app/",http.FileServer(http.Dir("."))))
	mux.Handle("assets/logo.png",http.FileServer(http.Dir("./assets/logo.png")))
	mux.HandleFunc("/healthz",func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-type","text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	corsMux := middlewareCors(mux)
	srv := &http.Server {
		Addr: "localhost:8080",
		Handler: corsMux,
	}
	log.Fatal(srv.ListenAndServe())
}
