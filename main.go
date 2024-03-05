package main

import (
	"encoding/json"
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
func postChirp(w http.ResponseWriter,r *http.Request) {
	type parameter struct {
		Body string `json:"body"`
	}
	type returnVals struct {
		Error string `json:"error,omitempty"`
		Valid bool `json:"valid,omitempty"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameter{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameter %s", err)
		w.WriteHeader(500)
		return
	}
	if len(params.Body) > 140 {
		resp := returnVals {
			Error: "Chirp is too long",
		}
		dat, err := json.Marshal(resp)
		if err != nil {
			log.Printf("Error unmarshalling JSON %s",err)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-type","application/json")
		w.WriteHeader(400)
		w.Write(dat)
	}else{resp := returnVals {
		Valid: true,
		}
		dat,err := json.Marshal(resp)
		if err != nil {
			log.Printf("Error unmarshalling JSON: %s",err)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-type","application/json")
		w.WriteHeader(200)
		w.Write(dat)
		}
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
	apirouter.Post("/validate_chirp",postChirp)
	log.Fatal(srv.ListenAndServe())
}
