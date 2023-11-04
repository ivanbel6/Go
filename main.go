package main

import (
	"fmt"
	"net/http"
)

type User struct {
	name                  string
	age                   uint16 //целое число это не может быть отрицательным
	money                 int16
	avg_grades, happiness float64
}

func (u User) getAllInfo() string {
	return fmt.Sprintf("User name is: %s. He is %d", u.name, u.age)
}
func (u *User) setNewName(newName string) {
	u.name = newName
}
func home_page(w http.ResponseWriter, r *http.Request) {
	bob := User{"Bob", 25, -50, 4.2, 0.8}
	bob.setNewName("Ivan")
	fmt.Fprintf(w, bob.getAllInfo())

}

func contacks_page(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Contcks page is here")
}

func handleRequest() {
	http.HandleFunc("/", home_page)
	http.HandleFunc("/contacts/", contacks_page)
	http.ListenAndServe(":8080", nil)
}

func main() {

	handleRequest()
}
