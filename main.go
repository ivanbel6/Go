package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"html/template"
	"io/ioutil"
	"net/http"
)

func main() {
	http.HandleFunc("/", handleFileUpload)
	http.ListenAndServe(":8080", nil)
}

func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Отправка HTML формы для загрузки файла
		tmpl := template.Must(template.ParseFiles("upload.html"))
		tmpl.Execute(w, nil)
	} else if r.Method == http.MethodPost {
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		data, err := ioutil.ReadAll(file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Подключение к базе данных
		clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
		client, err := mongo.Connect(context.Background(), clientOptions)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		collection := client.Database("Go_DB").Collection("Files")
		_, err = collection.InsertOne(context.Background(), bson.D{{"data", data}})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintln(w, "Файл успешно загружен!")
	} else {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
	}
}
