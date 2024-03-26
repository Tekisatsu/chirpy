package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"github.com/tekisatsu/chirpy/internal/database"
)
type Server struct {
	DB *database.DB
	apiConfig apiConfig
}
type apiConfig struct {
	fileserverHits int
	jwtSecret []byte
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
func (cfg *apiConfig) createAccessToken (id int) (string,error) {
	var expirationSeconds int = 3600 //one hour
	claims := &jwt.RegisteredClaims{
		Subject: strconv.Itoa(id),
		IssuedAt: jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Duration(expirationSeconds)*time.Second)),
		Issuer: "chirpy-access",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,claims)
	signedToken,err := token.SignedString(cfg.jwtSecret)
	if err != nil {
		return "",err
	}
	return signedToken,nil
}
func (cfg *apiConfig) createRefreshToken (id int) (string,error) {
	var expirationSeconds int = 86400*60 //60 days
	claims := &jwt.RegisteredClaims{
		Subject: strconv.Itoa(id),
		IssuedAt: jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Duration(expirationSeconds)*time.Second)),
		Issuer: "chirpy-refresh",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,claims)
	signedToken,err := token.SignedString(cfg.jwtSecret)
	if err != nil {
		return "",err
	}
	return signedToken,nil
}
func (cfg *apiConfig)validateToken(tokenStr string) (*jwt.Token,error) {
	claims := &jwt.RegisteredClaims{}
	var secretKey = cfg.jwtSecret
	token,err := jwt.ParseWithClaims(tokenStr,claims,func(token *jwt.Token)(interface{},error){
		return secretKey,nil
	})
	if err != nil {
		return nil,err
	}
	if !token.Valid {
		return nil, errors.New("Expired token")
	}
	return token,nil
}
func (s *Server)tokenRefresh(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")	
	tokenStr := strings.TrimPrefix(authHeader,"Bearer ")
	token,err := s.apiConfig.validateToken(tokenStr)
	if err != nil {
		log.Printf("Invalid token: %e",err)
		w.WriteHeader(401)
		return
	} else {
		claims,ok := token.Claims.(*jwt.RegisteredClaims)
		if !ok {
			log.Printf("Error getting Claims")
			w.WriteHeader(500)
			return
		}
		if claims.Issuer != "chirpy-refresh" {
			log.Printf("Invalid issuer")
			w.WriteHeader(401)
			return
		}
		err := s.DB.RefreshToken(tokenStr)
		if err != nil {
			log.Printf("Invalid token: %v",err)
			w.WriteHeader(401)
			return
		}
		id,err := strconv.Atoi(claims.Subject)
		if err != nil {
			log.Printf("Error getting id: %v",err)
			w.WriteHeader(500)
			return
		}
		newToken,err := s.apiConfig.createAccessToken(id)
		if err != nil {
			log.Printf("Error creating token: %v",err)
			w.WriteHeader(500)
			return
		}
		result := struct{
			AccessToken string `json:"token"`
		}{AccessToken: newToken}
		dat, err := json.Marshal(result)
		if err != nil {
			log.Printf("Error marshalling JSON: %v",err)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-type","Application/json")
		w.WriteHeader(200)
		w.Write(dat)
	}
}
func (s *Server)revokeToken(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")	
	tokenStr := strings.TrimPrefix(authHeader,"Bearer ")
	token,err := s.apiConfig.validateToken(tokenStr)
	if err != nil {
		log.Printf("Invalid token: %e",err)
		w.WriteHeader(401)
		return
	} else {
		claims,ok := token.Claims.(*jwt.RegisteredClaims)
		if !ok {
			log.Printf("Error getting Claims")
			w.WriteHeader(500)
			return
		}
		if claims.Issuer != "chirpy-refresh" {
			log.Printf("Invalid issuer")
			w.WriteHeader(401)
			return
		}
		err := s.DB.RevokeRefreshToken(tokenStr)
		if err != nil {
			log.Printf("Error revoking token: %v",err)
			w.WriteHeader(401)
			return
		}
		w.WriteHeader(200)
	}
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
func (s *Server)updateUsers(w http.ResponseWriter, r *http.Request){
	type parameter struct{
		Email string `json:"email"`
		Password string `json:"password"`
	}
	params := parameter{}
	authHeader := r.Header.Get("Authorization")
	tokenStr := strings.TrimPrefix(authHeader,"Bearer ")
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding params: %v",err)
		w.WriteHeader(500)
		return
	}
	token,err := s.apiConfig.validateToken(tokenStr)
	if err != nil {
		log.Printf("Invalid token: %e",err)
		w.WriteHeader(401)
		return
	} else {
		claims,ok := token.Claims.(*jwt.RegisteredClaims)
		if !ok {
			log.Printf("Error getting Claims")
			w.WriteHeader(500)
			return
		}
		if claims.Issuer != "chirpy-access" {
			log.Printf("Invalid issuer")
			w.WriteHeader(401)
			return
			}
		id,err := strconv.Atoi(claims.Subject)
		if err != nil {
			log.Printf("Error converting subject to id: %e",err)
			w.WriteHeader(500)
			return
		}
		updatedUser,errU :=s.DB.UpdateUser(params.Email,params.Password,id)
		if errU != nil {
			log.Printf("Error updating user: %e",errU)
			w.WriteHeader(500)
			return
		}
		dat,errM := json.Marshal(updatedUser)
		if errM != nil {
			log.Printf("Error marshalling JSON: %e",errM)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-type","application/json")
		w.WriteHeader(200)
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
	}else{
		valid,errV := s.DB.UserLogin(params.Email,params.Password)
		if errV != nil {
			log.Printf("Error validating: %v",errV)
			w.WriteHeader(401)
			return
		}
		accessToken,err := s.apiConfig.createAccessToken(valid.Id)
		refreshToken, err := s.apiConfig.createRefreshToken(valid.Id)
		if err != nil {
			log.Printf("Error creating token: %v",err)
			w.WriteHeader(500)
			return
		}
		type resp struct {
			AccessToken string `json:"token"`
			RefreshToken string `json:"refresh_token"`
			Email string `json:"email"`
			Id int `json:"id"`
		}
		respToken:= resp{
			AccessToken: accessToken,
			RefreshToken: refreshToken,
			Email: valid.Email,
			Id: valid.Id,
		}
		dat,errM := json.Marshal(respToken)
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
func (s *Server)deleteChirps(w http.ResponseWriter, r *http.Request){
	authHeader := r.Header.Get("Authorization")	
	tokenStr := strings.TrimPrefix(authHeader,"Bearer ")
	token,err := s.apiConfig.validateToken(tokenStr)
	if err != nil {
		log.Printf("Invalid token: %e",err)
		w.WriteHeader(401)
		return
	} else {
		idParam,errC := strconv.Atoi(chi.URLParam(r,"id"))
		if errC != nil {
			log.Printf("Error converting URLParam to int: %v",errC)
			return
		}
		claims,ok := token.Claims.(*jwt.RegisteredClaims)
		if !ok {
			log.Printf("Error getting Claims")
			w.WriteHeader(500)
			return
		}
		authorId,errI := strconv.Atoi(claims.Subject)
		if errI != nil {
			log.Printf("Error getting author id: %v",err)
			w.WriteHeader(500)
			return
		}
		errD := s.DB.DeleteChirp(idParam,authorId)
		if errD != nil {
			log.Printf("Error deleting Chirp: %e",errD)
			w.WriteHeader(403)
			return
		}
		w.WriteHeader(200)
		return
	}
}
func (s *Server)postChirps(w http.ResponseWriter, r *http.Request) {
	type parameter struct {
		Body string `json:"body"`
	}
	authHeader := r.Header.Get("Authorization")	
	tokenStr := strings.TrimPrefix(authHeader,"Bearer ")
	token,err := s.apiConfig.validateToken(tokenStr)
	if err != nil {
		log.Printf("Invalid token: %e",err)
		w.WriteHeader(401)
		return
	} else {
		claims,ok := token.Claims.(*jwt.RegisteredClaims)
		if !ok {
			log.Printf("Error getting Claims")
			w.WriteHeader(500)
			return
		}
		authorId,errI := strconv.Atoi(claims.Subject)
		if errI != nil {
			log.Printf("Error getting author id: %v",err)
			w.WriteHeader(500)
			return
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
			newChirp,err := s.DB.CreateChirp(cf,authorId)
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
	}	}
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
	godotenv.Load()
	jwtsecret := []byte(os.Getenv("JWT_SECRET"))
	db, err := database.NewDb("database.json")
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v",err)
	}
	apiCfg := apiConfig{
		jwtSecret: jwtsecret,
	}
	server := &Server{
		DB: db,
		apiConfig: apiCfg,
	}
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
	apirouter.Put("/users",server.updateUsers)
	apirouter.Post("/login",server.userLogin)
	apirouter.Post("/refresh",server.tokenRefresh)
	apirouter.Post("/revoke",server.revokeToken)
	apirouter.Delete("/chirps/{id}",server.deleteChirps)
	log.Fatal(srv.ListenAndServe())
}
