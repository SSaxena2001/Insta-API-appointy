package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type User struct {
	Id       string `json:"id,omitempty" bson:"_id,omitempty"`
	Name     string `json:"name,omitempty" bson:"name,omitempty"`
	Email    string `json:"email,omitempty" bson:"email,omitempty"`
	Password string `json:"password,omitempty" bson:"password,omitempty"`
}

type Post struct {
	Id            string    `json:"id,omitempty" bson:"_id,omitempty"`
	Caption       string    `json:"caption,omitempty" bson:"caption,omitempty"`
	ImgURL        string    `json:"imgUrl,omitempty" bson:"imgUrl,omitempty"`
	PostTimestamp time.Time `json:"postTimestamp,omitempty" bson:"postTimestamp,omitempty"`
}

var client *mongo.Client
var mongo_uri string = "mongodb://localhost:27017"
var SECRET_KEY string = "password" //Replace with your secret key to encrypt the password

func main() {
	fmt.Println("Starting the application...")
	connect()
	handleRequest()
}

func connect() {
	clientOptions := options.Client().ApplyURI(mongo_uri)
	client, _ = mongo.NewClient(clientOptions)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := client.Connect(ctx)

	if err != nil {
		log.Fatal(err)
	}
	err = client.Ping(context.Background(), readpref.Primary())

	if err != nil {
		log.Fatal("Couldn't connect to the database", err)
	} else {
		log.Println("Connected to MondoDB Server")
	}

}

//HomePage
func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!")
}

//Creating A New User
func createUser(res http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(`{"message":"GET METHOD is not applicable"}`))
	} else {
		if req.Header.Get("Content-Type") != "application/json" {
			res.WriteHeader(http.StatusInternalServerError)
			res.Write([]byte(`{"message":"Only json data is allowed"}`))
		} else {
			req.ParseForm()
			decoder := json.NewDecoder(req.Body)
			var newUser User
			err := decoder.Decode(&newUser)
			if err != nil {
				panic(err)
			}
			log.Println(newUser.Id)
			insertUser(newUser)
		}
	}
}

//Using To Insert The NewUser
func insertUser(user User) {
	users := client.Database("InstaDB").Collection("users")
	user.Password = string(encrypt([]byte(user.Password), SECRET_KEY))
	insertResult, err := users.InsertOne(context.TODO(), user)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Inserted User with Id: ", insertResult.InsertedID)
}

//Encrypting password by hashing it
func encrypt(data []byte, passphrase string) []byte {
	block, _ := aes.NewCipher([]byte(createHash(passphrase)))
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext
}

//creating Hash of the password
func createHash(key string) string {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

func getUserById(res http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	id := strings.TrimPrefix(req.URL.Path, "/users/")
	fmt.Println("User Id:", id)
	var user User
	users := client.Database("InstaDB").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := users.FindOne(ctx, User{Id: id}).Decode(&user)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(`{"message":"` + err.Error() + `"}`))
		return
	}
	fmt.Println("Returned User Id: ", user.Id)
	json.NewEncoder(res).Encode(user)
}

//Creating A New Post
func createPost(res http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(`{"message":"GET METHOD is not applicable"}`))
	} else {
		if req.Header.Get("Content-Type") != "application/json" {
			res.WriteHeader(http.StatusInternalServerError)
			res.Write([]byte(`{"message":"Only json data is allowed"}`))
		} else {
			req.ParseForm()
			decoder := json.NewDecoder(req.Body)
			var newPost Post
			newPost.PostTimestamp = time.Now()
			err := decoder.Decode(&newPost)
			if err != nil {
				panic(err)
			}
			log.Println(newPost.Id)
			insertPost(newPost)
		}
	}
}

//Using To Insert The NewPost
func insertPost(post Post) {
	posts := client.Database("InstaDB").Collection("posts")
	result, err := posts.InsertOne(context.TODO(), post)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Inserted Post with Id: ", result.InsertedID)
}

//Getting post by id
func getPostById(res http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	id := strings.TrimPrefix(req.URL.Path, "/posts/")
	fmt.Println("Post Id:", id)
	var post Post
	posts := client.Database("InstaDB").Collection("posts")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := posts.FindOne(ctx, Post{Id: id}).Decode(&post)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(`{"message":"` + err.Error() + `"}`))
		return
	}
	fmt.Println("Returned Post Id: ", post.Id)
	json.NewEncoder(res).Encode(post)
}

func handleRequest() {

	http.HandleFunc("/", homePage)
	http.HandleFunc("/users", createUser)
	http.HandleFunc("/users/", getUserById)
	http.HandleFunc("/posts", createPost)
	http.HandleFunc("/posts/", getPostById)
	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		log.Fatal("ListenAndServe", err)
	}
}
