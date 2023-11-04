package main

import (
	"context"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"html/template"
	"io/ioutil"
	"net/http"
)

var collection *mongo.Collection

func main() {
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		fmt.Println("Ошибка при подключении к базе данных:", err)
		return
	}
	defer client.Disconnect(context.Background())

	collection = client.Database("Go_DB").Collection("Files")

	http.HandleFunc("/", handleFileUpload)
	http.HandleFunc("/files/", handleFileDelete)
	http.HandleFunc("/info", handleFileList)
	http.HandleFunc("/specific/", handleSpecificFile)

	http.ListenAndServe(":8080", nil)
}

func handleSpecificFile(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		id := r.URL.Path[len("/specific/"):]
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			http.Error(w, "Недопустимый id файла", http.StatusBadRequest)
			return
		}

		var result bson.M
		err = collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&result)
		if err != nil {
			http.Error(w, "Файл не найден", http.StatusNotFound)
			return
		}

		data, ok := result["data"].(primitive.Binary)
		if !ok {
			http.Error(w, "Данные файла недоступны", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Disposition", "attachment; filename="+id)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(data.Data)
	} else {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
	}
}

func handleFileList(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		cursor, err := collection.Find(context.Background(), bson.D{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer cursor.Close(context.Background())

		var files []string
		for cursor.Next(context.Background()) {
			var result bson.M
			if err := cursor.Decode(&result); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			files = append(files, result["_id"].(primitive.ObjectID).Hex())
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(files)
	} else {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
	}
}

func handleFileDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		id := r.URL.Path[len("/files/"):]
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			http.Error(w, "Недопустимый id файла", http.StatusBadRequest)
			return
		}

		result, err := collection.DeleteOne(context.Background(), bson.M{"_id": objectID})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if result.DeletedCount == 0 {
			http.Error(w, "Файл не найден", http.StatusNotFound)
			return
		}

		fmt.Fprintln(w, "Файл успешно удален!")
	} else {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
	}
}

func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		cursor, err := collection.Find(context.Background(), bson.D{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer cursor.Close(context.Background())

		var files []string
		for cursor.Next(context.Background()) {
			var result bson.M
			if err := cursor.Decode(&result); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			files = append(files, result["_id"].(primitive.ObjectID).Hex())
		}

		tmpl := template.Must(template.ParseFiles("upload.html"))
		tmpl.Execute(w, struct{ Files []string }{files})
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
