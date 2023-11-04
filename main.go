package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func SetUserDataHandler(w http.ResponseWriter, r *http.Request) {
	user := User{
		ID:   "666",
		Name: "IVAN_WAS_HERE",
	}

	data, err := json.Marshal(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	encodedData := base64.StdEncoding.EncodeToString(data)
	cookie := &http.Cookie{
		Name:  "userdata",
		Value: encodedData,
		Path:  "/",
	}
	http.SetCookie(w, cookie)

	w.Write([]byte("User data saved in cookie"))
}

func GetUserDataHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("userdata")
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	encodedData := cookie.Value

	// Линейная версия
	startTime := time.Now()
	data, err := base64.StdEncoding.DecodeString(encodedData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var user User
	err = json.Unmarshal(data, &user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
	log.Printf("Linear processing took %s\n", time.Since(startTime))

	// Конкурентная версия
	startTime = time.Now()
	ch := make(chan []byte)
	go func() {
		data, err := base64.StdEncoding.DecodeString(encodedData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ch <- data
	}()

	go func() {
		var user User
		data := <-ch
		err := json.Unmarshal(data, &user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonData, err := json.MarshalIndent(user, "", "  ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
		log.Printf("Concurrent processing took %s\n", time.Since(startTime))
	}()

}

func main() {
	logFile, err := os.OpenFile("error.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()

	log.SetOutput(io.MultiWriter(os.Stdout, logFile))

	router := mux.NewRouter()
	router.HandleFunc("/api/setdata", SetUserDataHandler).Methods("POST")
	router.HandleFunc("/api/getdata", GetUserDataHandler).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s", port)
	err = http.ListenAndServe(":"+port, router)
	if err != nil {
		log.Fatal(err)
	}
}
