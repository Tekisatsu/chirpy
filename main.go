package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"github.com/go-chi/chi/v5"
	"github.com/tekisatsu/chirpy/internal/database"
)
type Server struct {
	DB *database.DB
}
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
func (s *Server)createUser(w http.ResponseWriter, r *http.Request) {
	type parameter struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	params := parameter{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding params: %v",err)
		w.WriteHeader(500)
		return
	} else {
		newUser,err := s.DB.CreateUser(params.Email,params.Password)
		if err != nil {
			log.Printf("Error creating user: %v",err)
			w.WriteHeader(500)
			return
		}
		dat,errM := json.Marshal(newUser)
		if errM != nil {
			log.Printf("Error marshalling user: %v",errM)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-type","application/json")
		w.WriteHeader(201)
		w.Write(dat)
	}
	
}
func (s *Server)userLogin(w http.ResponseWriter, r *http.Request) {
	type parameter struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	params := parameter{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding params: %v",err)
		w.WriteHeader(500)
		return
	}else{valid,errV := s.DB.UserLogin(params.Email,params.Password)
		if errV != nil {
			log.Printf("Error validating: %v",errV)
			w.WriteHeader(401)
			return
		}
		dat,errM := json.Marshal(valid)
		if errM != nil {
			log.Printf("Error Marshalling JSON: %v",errM)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-type","application/json")
		w.WriteHeader(200)
		w.Write(dat)
	}
}
func (s *Server)postChirps(w http.ResponseWriter, r *http.Request) {
	type parameter struct {
		Body string `json:"body"`
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	params := parameter{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameter %s", err)
		w.WriteHeader(500)
		return
	}
	if len(params.Body) > 140 {
		dat, err := json.Marshal("Error: Chirp too long")
		if err != nil {
			log.Printf("Error marshalling JSON %s",err)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-type","application/json")
		w.WriteHeader(400)
		w.Write(dat)
	}else{
		cf := chirpFilter(&params.Body)
		newChirp,err := s.DB.CreateChirp(cf)
		if err != nil {
			log.Printf("Error creating Chirp: %s",err)
			w.WriteHeader(500)
			return
		}
		dat,err := json.Marshal(newChirp)
		if err != nil {
			log.Printf("Error marshaling json: %v",err)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-type","application/json")
		w.WriteHeader(201)
		w.Write(dat)
		}
}
func chirpFilter (msg *string) string {
	splitMsg := strings.Split(*msg," ")
	for i,word := range splitMsg {
		switch {
		case strings.ToLower(word) == "kerfuffle": splitMsg[i]="****"
		case strings.ToLower(word) == "sharbert": splitMsg[i]="****"
		case strings.ToLower(word) == "fornax": splitMsg[i]="****"
		}
	}
	return strings.Join(splitMsg," ")
}
func (s *Server) getChirps (w http.ResponseWriter,r *http.Request) {
	chirps,err := s.DB.GetChirps()
	if err != nil {
		log.Printf("Error getting Chirps: %v",err)
		w.WriteHeader(500)
		return
	}
	dat,errM := json.Marshal(chirps)
	if errM != nil {
		log.Printf("Error marshalling Chirps: %v",errM)
		return
	}
	w.Header().Set("Content-type","application/json")
	w.WriteHeader(200)
	w.Write(dat)
}
func (s *Server) getChirp (w http.ResponseWriter,r *http.Request) {
	idParam,errC := strconv.Atoi(chi.URLParam(r,"id"))
	if errC != nil {
		log.Printf("Error converting URLParam to int: %v",errC)
		return
	}
	chirp,err := s.DB.GetChirp(idParam)
	if err != nil {
		log.Printf("%v",err)
		w.WriteHeader(404)
		return
	}
	dat,errM := json.Marshal(chirp)
	if errM != nil {
		log.Printf("Error mashalling Chirp: %v",err)
		return
	}
	w.Header().Set("Content-type","Application/json")
	w.WriteHeader(200)
	w.Write(dat)
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
	db, err := database.NewDb("database.json")
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v",err)
	}
	server := &Server{
		DB: db,
	}
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
	apirouter.Post("/chirps",server.postChirps)
	apirouter.Get("/chirps",server.getChirps)
	apirouter.Get("/chirps/{id}",server.getChirp)
	apirouter.Post("/users",server.createUser)
	apirouter.Post("/login",server.userLogin)
	log.Fatal(srv.ListenAndServe())
}
