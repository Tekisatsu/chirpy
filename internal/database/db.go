package database

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type DB struct {
	path string
	mux *sync.RWMutex
}
type DBSuper struct {
	DBStructure DBStructure
	UserInternal []UserInternal
	RevokedTokens map[string]time.Time
}
type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
	User UserResponse
	UserAmount int `json:"usreamount"`
}
type Chirp struct {
	Id int `json:"id"`
	Body string `json:"body"`
	AuthorId int `json:"author_id"`
}
type UserResponse struct {
	Id int `json:"id"`
	Email string `json:"email"`
}
type UserInternal struct {
	Id int `json:"id"`
	Email string `json:"email"`
	Password []byte `json:"password"`
}
func (db *DB) RefreshToken(tokenStr string) error {
	db.mux.Lock()
	defer db.mux.Unlock()
	dbSuper,err := db.loadDb()
	if err != nil {
		return err
	}
	if _,ok := dbSuper.RevokedTokens[tokenStr];ok {
		return errors.New("Revoked token")
	}
	return nil
}
func (db *DB) RevokeRefreshToken (tokenStr string)error{
	db.mux.Lock()
	defer db.mux.Unlock()
	dbSuper,err := db.loadDb()
	if err != nil {
		return err
	}
	dbSuper.RevokedTokens[tokenStr]=time.Now().UTC()
	dat,err := json.Marshal(dbSuper)
	if err != nil {
		return err
	}
	os.WriteFile(db.path,dat,0600)
	return nil
}
func (db *DB) createUserPassword (pword string)([]byte,error) {
	hash,err := bcrypt.GenerateFromPassword([]byte(pword),14)
	if err != nil {
		return nil,err
	}
	return hash,nil
}
func (db *DB)CreateUser(email,password string)(UserResponse,error){
	db.mux.Lock()
	defer db.mux.Unlock()
	dbSuper,err := db.loadDb()
	if err != nil {
		return UserResponse{},err
	}
	for _,user := range dbSuper.UserInternal {
		if user.Email == email {
			return UserResponse{},errors.New("Email already in use")
		}
	}
	maxId:= dbSuper.DBStructure.UserAmount+1
	dbSuper.DBStructure.UserAmount = maxId
	pWord,errP := db.createUserPassword(password)
	if errP != nil {
		return UserResponse{},errP
	}
	newUser := UserResponse{
		Email: email,
		Id: maxId,
	}
	newInternalUser := UserInternal{
		Email: email,
		Id: maxId,
		Password: pWord,
	}
	dbSuper.UserInternal = append(dbSuper.UserInternal, newInternalUser)
	dat,errM := json.Marshal(dbSuper)
	if errM != nil {
		return UserResponse{},errM
	}
	errW := os.WriteFile(db.path,dat,0600)
	if errW != nil {
		return UserResponse{},errW
	}
	return newUser,nil
}
func (db *DB) UserLogin (email,password string) (UserResponse,error) {
	db.mux.Lock()
	defer db.mux.Unlock()
	dbSuper,err := db.loadDb()
	if err != nil {
		return UserResponse{},err
	}
	for _,user := range dbSuper.UserInternal {
		if user.Email == email {
			err := bcrypt.CompareHashAndPassword(user.Password, []byte(password))
			if err != nil {
				return UserResponse{},errors.New("Invalid information")
			} else {
				return struct{
					Id int `json:"id"`
					Email string `json:"email"`
				}{Id:user.Id, Email: user.Email},nil
			}
		}	
	}
	return UserResponse{},errors.New("Invalid information")
} 
func (db *DB)UpdateUser(email,password string, id int)(UserResponse,error){
	db.mux.Lock()
	defer db.mux.Unlock()
	dbSuper,err := db.loadDb()
	if err != nil {
		return UserResponse{}, err
	}
	var updatedUser UserResponse
	for i,user := range dbSuper.UserInternal {
		if user.Id==id {
			newPw,err := db.createUserPassword(password)
			if err != nil {
				return UserResponse{}, nil
			}
			dbSuper.UserInternal[i].Email = email
			dbSuper.UserInternal[i].Password = newPw
			updatedUser = UserResponse{
				Id: user.Id,
				Email: email,
			}
		}
	}
	dat,err := json.Marshal(dbSuper)
	if err != nil {
		return UserResponse{},err
	}
	errW := os.WriteFile(db.path,dat,0600)
	if errW != nil {
		return UserResponse{}, errW
	}
	return updatedUser,nil
}
func (db *DB)CreateChirp(body string,authorId int)(Chirp,error){
	db.mux.Lock()
	defer db.mux.Unlock()
	dbSuper,err := db.loadDb()
	if err != nil {
		return Chirp{},err
	}
	var maxId int
	for id := range dbSuper.DBStructure.Chirps{
		if id > maxId {
			maxId=id
		}
	}
	maxId++
	newChirp := Chirp{
		Id: maxId,
		Body: body,
		AuthorId: authorId,
	}
	dbSuper.DBStructure.Chirps[newChirp.Id]=newChirp
	dat,errM := json.Marshal(dbSuper)
	if errM != nil {
		return Chirp{},errM
	}
	errW:=os.WriteFile(db.path,dat,0600)
	if errW != nil {
		return Chirp{},errW
	}
	return newChirp,nil
}
func (db *DB)loadDb()(DBSuper,error){
	dbSuper := DBSuper{}
	data,errR := os.ReadFile(db.path)
	if errR != nil {
		return DBSuper{},errR
	}
	errU := json.Unmarshal(data,&dbSuper)
	if errU != nil {
		return DBSuper{},errU
	}
	return dbSuper,nil
}
func (db *DB) GetChirps () ([]Chirp,error) {
	db.mux.Lock()
	defer db.mux.Unlock()
	dbstructure,err:=db.loadDb()
	if err != nil {
		return nil,err
	}
	chirps := make([]Chirp ,len(dbstructure.DBStructure.Chirps))
	i := 0
	for _,chirp:= range dbstructure.DBStructure.Chirps {
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
	dbSuper,err := db.loadDb()
	if err != nil {
		return Chirp{},err
	}
	if val,ok := dbSuper.DBStructure.Chirps[id];ok {
		return val,nil
	}
	return Chirp{},errors.New("Chirp not found")
}
func NewDb(path string) (*DB, error) {
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			initialData :=&DBSuper{ 
				DBStructure: DBStructure{
					Chirps: map[int]Chirp{},
			},RevokedTokens: make(map[string]time.Time),
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
