package database

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"sync"
)

type DB struct {
	path string
	mux *sync.RWMutex
}
type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
	User User
	UserAmount int `json:"usreamount"`
}
type Chirp struct {
	Id int `json:"id"`
	Body string `json:"body"`
}
type User struct {
	Id int `json:"id"`
	Email string `json:"email"`
}
func (db *DB)CreateUser(email string)(User,error){
	db.mux.Lock()
	defer db.mux.Unlock()
	dbstucture,err := db.LoadDb()
	if err != nil {
		return User{},err
	}
	maxId:=dbstucture.UserAmount+1
	dbstucture.UserAmount = maxId
	newUser := User{
		Email: email,
		Id: maxId,
	}
	dat,errM := json.Marshal(dbstucture)
	if errM != nil {
		return User{},errM
	}
	errW := os.WriteFile(db.path,dat,0600)
	if errW != nil {
		return User{},errW
	}
	return newUser,nil
}
func (db *DB)CreateChirp(body string)(Chirp,error){
	db.mux.Lock()
	defer db.mux.Unlock()
	dbstructure,err := db.LoadDb()
	if err != nil {
		return Chirp{},err
	}
	var maxId int
	for id := range dbstructure.Chirps{
		if id > maxId {
			maxId=id
		}
	}
	maxId++
	newChirp := Chirp{
		Id: maxId,
		Body: body,
	}
	dbstructure.Chirps[newChirp.Id]=newChirp
	dat,errM := json.Marshal(dbstructure)
	if errM != nil {
		return Chirp{},errM
	}
	errW:=os.WriteFile(db.path,dat,0600)
	if errW != nil {
		return Chirp{},errW
	}
	return newChirp,nil
}
func (db *DB)LoadDb()(DBStructure,error){
	dbStructure := DBStructure{}
	data,errR := os.ReadFile(db.path)
	if errR != nil {
		return DBStructure{},errR
	}
	errU := json.Unmarshal(data,&dbStructure)
	if errU != nil {
		return DBStructure{},errU
	}
	return dbStructure,nil
}
func (db *DB) GetChirps () ([]Chirp,error) {
	db.mux.Lock()
	defer db.mux.Unlock()
	dbstructure,err:=db.LoadDb()
	if err != nil {
		return nil,err
	}
	chirps := make([]Chirp ,len(dbstructure.Chirps))
	i := 0
	for _,chirp:= range dbstructure.Chirps {
		chirps[i] = chirp
		i++
	}
	sort.SliceStable(chirps,func(i, j int) bool {
		return chirps[i].Id < chirps[j].Id
	})
	return chirps,nil
}
func (db *DB) GetChirp (id int) (Chirp,error) {
	db.mux.Lock()
	defer db.mux.Unlock()
	dbStructure,err := db.LoadDb()
	if err != nil {
		return Chirp{},err
	}
	if val,ok := dbStructure.Chirps[id];ok {
		return val,nil
	}
	return Chirp{},errors.New("Chirp not found")
}
func NewDb(path string) (*DB, error) {
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			initialData := &DBStructure{
				Chirps: map[int]Chirp{},
			}
			data, err := json.Marshal(initialData)
			if err != nil {
				return nil, err
			}
			err = os.WriteFile(path, data, 0600)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return &DB{
		path: path,
		mux:  &sync.RWMutex{},
	}, nil
}
