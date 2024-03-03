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
	apirouter := chi.NewRouter()
	adminrouter := chi.NewRouter()
	r.Mount("/api",apirouter)
	r.Mount("/admin",adminrouter)
	r.Handle("/app",apiCfg.hitsCounter(http.StripPrefix("/app",http.FileServer(http.Dir(".")))))
	r.Handle("/app/*",apiCfg.hitsCounter(http.StripPrefix("/app",http.FileServer(http.Dir(".")))))
	r.Handle("/assets/logo.png",apiCfg.hitsCounter(http.FileServer(http.Dir("./assets/logo.png"))))
	apirouter.Get("/healthz",func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-type","text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	adminrouter.Get("/metrics", func(w http.ResponseWriter, r *http.Request){
		w.Header().Set("Content-type","text/html")
		w.Write([]byte(fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>",apiCfg.fileserverHits)))
	})
	apirouter.Handle("/reset", apiCfg.resetHitsCounter())
	corsMux := middlewareCors(r)
	srv := &http.Server {
		Addr: "localhost:8080",
		Handler: corsMux,
	}
	log.Fatal(srv.ListenAndServe())
}
