package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	bucket *gridfs.Bucket
	client *mongo.Client
)

func main() {
	var err error

	client, err = mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}

	bucket, err = gridfs.NewBucket(
		client.Database("Go_DB"),
	)
	if err != nil {
		log.Println(err)
		return
	}

	http.HandleFunc("/upload", handleFileUpload)
	http.HandleFunc("/download/", handleFileDownload)
	http.HandleFunc("/delete/", handleFileDelete)
	http.HandleFunc("/files", handleFileList)

	http.HandleFunc("/file/", handleFileGetInfo)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleFileGetInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET is supported", http.StatusMethodNotAllowed)
		return
	}

	fileID := strings.TrimPrefix(r.URL.Path, "/file/")
	objectID, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		http.Error(w, "Invalid file ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var fileInfo bson.M
	err = client.Database("Go_DB").Collection("fs.files").FindOne(ctx, bson.M{"_id": objectID}).Decode(&fileInfo)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	fmt.Fprintf(w, "File ID: %s, File Name: %s, File Size: %d, Upload Date: %v\n", fileInfo["_id"], fileInfo["filename"], fileInfo["length"], fileInfo["uploadDate"])
}

func handleFileList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET is supported", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Используйте глобальную переменную client
	coll := client.Database("Go_DB").Collection("fs.files")

	cursor, err := coll.Find(ctx, bson.M{})
	if err != nil {
		http.Error(w, "Error while retriving files", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		http.Error(w, "Error while parsing files", http.StatusInternalServerError)
		return
	}

	for _, result := range results {
		fmt.Fprintf(w, "File Name: %s, File Size: %d, Upload Date: %v\n", result["filename"], result["length"], result["uploadDate"])
	}
}

func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST is supported", http.StatusMethodNotAllowed)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	uploadStream, err := bucket.OpenUploadStream(header.Filename, &options.UploadOptions{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer uploadStream.Close()

	_, err = io.Copy(uploadStream, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "File uploaded successfully with ID: %s", uploadStream.FileID)
}

func handleFileDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET is supported", http.StatusMethodNotAllowed)
		return
	}

	fileID := strings.TrimPrefix(r.URL.Path, "/download/")
	objectID, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		http.Error(w, "Invalid file ID", http.StatusBadRequest)
		return
	}

	downloadStream, err := bucket.OpenDownloadStream(objectID)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer downloadStream.Close()

	_, err = io.Copy(w, downloadStream)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleFileDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Only DELETE is supported", http.StatusMethodNotAllowed)
		return
	}

	fileID := strings.TrimPrefix(r.URL.Path, "/delete/")
	objectID, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		http.Error(w, "Invalid file ID", http.StatusBadRequest)
		return
	}

	err = bucket.Delete(objectID)
	if err != nil {
		http.Error(w, "File not found or error while deleting", http.StatusInternalServerError)
		return
	}

	fmt.Fprintln(w, "File deleted successfully")
}
