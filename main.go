package main

import (
	"fmt"
	"log"
	"net/http"
	"github.com/go-chi/chi/v5"
)
	
type apiConfig struct {
	fileserverHits int
}
func (cfg *apiConfig) hitsCounter (next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits ++
		next.ServeHTTP(w,r)
	})
}
func (cfg *apiConfig) resetHitsCounter () http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		cfg.fileserverHits = 0
	}) 
}
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
	apiCfg := apiConfig{}
	r := chi.NewRouter()
	r.Handle("/app",apiCfg.hitsCounter(http.StripPrefix("/app",http.FileServer(http.Dir(".")))))
	r.Handle("/app/*",apiCfg.hitsCounter(http.StripPrefix("/app",http.FileServer(http.Dir(".")))))
	r.Handle("/assets/logo.png",apiCfg.hitsCounter(http.FileServer(http.Dir("./assets/logo.png"))))
	r.Get("/healthz",func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-type","text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request){
		w.Header().Set("Content-type","text/plain; charset=utf-8")
		w.Write([]byte(fmt.Sprintf("Hits: %v",apiCfg.fileserverHits)))
	})
	r.Handle("/reset", apiCfg.resetHitsCounter())
	corsMux := middlewareCors(r)
	srv := &http.Server {
		Addr: "localhost:8080",
		Handler: corsMux,
	}
	log.Fatal(srv.ListenAndServe())
}
