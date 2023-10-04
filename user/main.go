package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// User struct for storing user information
type User struct {
	Name     string
	Email    string
	Password string
}

// Book struct for storing book information
type Book struct {
	ID     primitive.ObjectID `bson:"_id,omitempty"`
	Name   string             `bson:"name"`
	Author string             `bson:"author"`
	Cost   float64            `bson:"cost"`
}

// MongoDB configuration
const (
	ConnectionString = "mongodb://localhost:27017"
	DatabaseName      = "mydatabase"
	UsersCollection   = "users"
	BooksCollection   = "books"
)

var (
	client           *mongo.Client
	usersCollection  *mongo.Collection
	booksCollection  *mongo.Collection
)

func init() {
	// Create a MongoDB client
	clientOptions := options.Client().ApplyURI(ConnectionString)
	var err error
	client, err = mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	// Check the connection
	err = client.Ping(context.Background(), readpref.Primary())
	if err != nil {
		log.Fatal(err)
	}

	// Get handles to the users and books collections
	usersCollection = client.Database(DatabaseName).Collection(UsersCollection)
	booksCollection = client.Database(DatabaseName).Collection(BooksCollection)
}

func main() {
	http.HandleFunc("/", loginHandler)
	http.HandleFunc("/signup", signupHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/book", bookHandler)
	http.HandleFunc("/submit", submitBookHandler)
	http.HandleFunc("/modify", modifyBookHandler)
	http.HandleFunc("/delete", deleteBookHandler)

	fmt.Println("Server is running on http://localhost:8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}

func signupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		renderTemplate(w, "signup.html", nil)
	} else if r.Method == "POST" {
		// Get form values
		name := r.FormValue("name")
		email := r.FormValue("email")
		password := r.FormValue("password")

		// Create a new user object
		user := User{
			Name:     name,
			Email:    email,
			Password: password,
		}

		// Insert the user into the users collection
		_, err := usersCollection.InsertOne(context.Background(), user)
		if err != nil {
			http.Error(w, "Error creating user", http.StatusInternalServerError)
			return
		}

		// Redirect to login page
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		renderTemplate(w, "login.html", nil)
	} else if r.Method == "POST" {
		// Get form values
		email := r.FormValue("email")
		password := r.FormValue("password")

		// Check if user exists
		filter := bson.M{"email": email, "password": password}
		var user User
		err := usersCollection.FindOne(context.Background(), filter).Decode(&user)
		if err != nil {
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}

		// Redirect to book page
		http.Redirect(w, r, "/book", http.StatusSeeOther)
	}
}

func bookHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve books from the books collection
	cursor, err := booksCollection.Find(context.Background(), bson.M{})
	if err != nil {
		http.Error(w, "Failed to retrieve books", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var books []Book
	for cursor.Next(context.Background()) {
		var book Book
		if err := cursor.Decode(&book); err != nil {
			http.Error(w, "Failed to decode book", http.StatusInternalServerError)
			return
		}
		books = append(books, book)
	}

	// Render the book page template with the book data
	renderTemplate(w, "book.html", struct{ Books []Book }{books})
}

func submitBookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Retrieve form values
	name := r.FormValue("name")
	author := r.FormValue("author")
	cost := r.FormValue("cost")

	// Convert cost to float64
	bookCost, err := strconv.ParseFloat(cost, 64)
	if err != nil {
		http.Error(w, "Invalid cost", http.StatusBadRequest)
		return
	}

	// Create a Book instance
	book := Book{
		Name:   name,
		Author: author,
		Cost:   bookCost,
	}

	// Insert book into the books collection
	_, err = booksCollection.InsertOne(context.Background(), book)
	if err != nil {
		http.Error(w, "Failed to insert book", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/book", http.StatusSeeOther)
}

func modifyBookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bookID := r.FormValue("id")
	if bookID == "" {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	// Retrieve form values
	name := r.FormValue("name")
	author := r.FormValue("author")
	cost := r.FormValue("cost")

	// Convert cost to float64
	bookCost, err := strconv.ParseFloat(cost, 64)
	if err != nil {
		http.Error(w, "Invalid cost", http.StatusBadRequest)
		return
	}

	// Update book details in the books collection
	objID, err := primitive.ObjectIDFromHex(bookID)
	if err != nil {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	filter := bson.M{"_id": objID}
	update := bson.M{
		"$set": bson.M{
			"name":   name,
			"author": author,
			"cost":   bookCost,
		},
	}

	_, err = booksCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		http.Error(w, "Failed to update book", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/book", http.StatusSeeOther)
}

func deleteBookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bookID := r.FormValue("id")
	if bookID == "" {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	// Delete book from the books collection
	objID, err := primitive.ObjectIDFromHex(bookID)
	if err != nil {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	filter := bson.M{"_id": objID}
	_, err = booksCollection.DeleteOne(context.Background(), filter)
	if err != nil {
		http.Error(w, "Failed to delete book", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/book", http.StatusSeeOther)
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	tmpl = fmt.Sprintf("templates/%s", tmpl)
	t, err := template.ParseFiles(tmpl)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
