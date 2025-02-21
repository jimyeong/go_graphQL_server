package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"example.com/m/v2/helper"
	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
	"github.com/graphql-go/graphql"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
)

var user struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	Picture     string `json:"picture"`
	LocalId     string `json:"localId"`
	DisplayName string `json:"displayName"`
	DateOfBirth string `json:"dateOfBirth"`
	PhoneNumber string `json:"phoneNumber"`
	Verified    bool   `json:"verified"`
	Length      int    `json:"length"`
}

var queryType = graphql.NewObject(graphql.ObjectConfig{})

var schema, _ = graphql.NewSchema(graphql.SchemaConfig{})

func configSQLDB(context context.Context, DB_HOST string, DB_USER string, DB_PASSWORD string, DB_NAME string) (*sql.DB, error) {

	DB_CONNECTION := fmt.Sprintf("%s:%s@tcp(%s)/%s", DB_USER, DB_PASSWORD, DB_HOST, DB_NAME)
	fmt.Print(DB_CONNECTION)
	mysqlClient, err := sql.Open("mysql", DB_CONNECTION)
	if err != nil {
		panic(err)
	}
	return mysqlClient, err
}
func configRedis(context context.Context, DB_HOST string, DB_PASSWORD string) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     DB_HOST,
		Password: DB_PASSWORD, // no password set
		DB:       0,           // use default DB
	})

	err := rdb.Set(context, "key", "value", 0).Err()
	if err != nil {
		panic(err)
	}

	val, err := rdb.Get(context, "key").Result()
	if err != nil {
		panic(err)
	}
	fmt.Println("key", val)

	val2, err := rdb.Get(context, "key2").Result()
	if err == redis.Nil {
		fmt.Println("key2 does not exist")
	} else if err != nil {
		panic(err)
	} else {
		fmt.Println("key2", val2)
	}
	return rdb
	// Output: key value
	// key2 does not exist

}

// db connection
func configOauth(OAUTH_KEY string, OAUTH_SECRET string, OAUTH_REDIRECT_URL string, ORIGIN string) *oauth2.Config {
	conf := &oauth2.Config{
		ClientID:     OAUTH_KEY,
		ClientSecret: OAUTH_SECRET,
		Scopes:       []string{"email", "profile"},
		RedirectURL:  ORIGIN + OAUTH_REDIRECT_URL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}
	return conf
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	var OAUTH_KEY string = os.Getenv("GOOGLE_OAUTH_CLIENT_ID")
	var OAUTH_SECRET string = os.Getenv("GOOGLE_OAUTH_SECRET")
	var OAUTH_REDIRECT_URL string = os.Getenv("GOOGLE_OAUTH_REDIRECT_URL")
	var ORIGIN string = os.Getenv("ORIGIN")
	var MYSQL_HOST string = os.Getenv("MYSQL_HOST")
	var MYSQL_USER string = os.Getenv("MYSQL_USER")
	var MYSQL_PASSWORD string = os.Getenv("MYSQL_PASSWORD")
	var MYSQL_DB string = os.Getenv("MYSQL_DB")
	// var REDIS_HOST string = os.Getenv("REDIS_HOST")
	// var rdsClient *redis.Client = configRedis(context.Background(), REDIS_HOST, "")
	conf := configOauth(OAUTH_KEY, OAUTH_SECRET, OAUTH_REDIRECT_URL, ORIGIN)
	mysqlClient, err := configSQLDB(context.Background(), MYSQL_HOST, MYSQL_USER, MYSQL_PASSWORD, MYSQL_DB)
	if err != nil {
		log.Fatal(err)
	}
	defer mysqlClient.Close()

	// rdsClient.Set(context.Background(), "key", "value", 0)

	// csrf protection token generation
	randomStr := helper.RandomString(10)
	verifier := oauth2.GenerateVerifier()
	// router
	http.HandleFunc("/oauth/google", GoogleOauthLogin(conf, randomStr, verifier))
	http.HandleFunc(OAUTH_REDIRECT_URL, GoogleOauthCallback(conf, randomStr, verifier, mysqlClient))
	// start server
	log.Fatal(http.ListenAndServe(":3000", nil))
}
func GoogleOauthLogin(conf *oauth2.Config, randomStr string, verifier string) func(w http.ResponseWriter, r *http.Request) {
	url := conf.AuthCodeURL(randomStr, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	}
}
func GoogleOauthCallback(conf *oauth2.Config, randomStr string, verifier string, sqlClient *sql.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// csrf protection
		if r.FormValue("state") != randomStr {
			http.Error(w, "Invalid OAuth state", http.StatusBadRequest)
			return
		}
		code := r.FormValue("code")
		token, err := conf.Exchange(context.Background(), code, oauth2.VerifierOption(verifier))
		if err != nil {
			http.Error(w, "Code exchange failed", http.StatusInternalServerError)
			return
		}
		// Use token to get user information
		client := conf.Client(context.Background(), token)
		userInfo, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
		if err != nil {
			http.Error(w, "Failed to get user info", http.StatusInternalServerError)
			return
		}
		defer userInfo.Body.Close()
		if err := json.NewDecoder(userInfo.Body).Decode(&user); err != nil {
			http.Error(w, "Failed to decode user info", http.StatusInternalServerError)
			return
		}

		if user.Email == "" {
			http.Error(w, "Email is required", http.StatusBadRequest)
			return
		}

		/**

			ID          string `json:"id"`
			Email       string `json:"email"`
			Name        string `json:"name"`
			Picture     string `json:"picture"`
			LocalId     string `json:"localId"`
			DisplayName string `json:"displayName"`
			DateOfBirth string `json:"dateOfBirth"`
			PhoneNumber string `json:"phoneNumber"`
			Verified    bool   `json:"verified"`
		**/
		query := `
			INSERT INTO users (
				account_id,
				oauth_provider,
				oauth_account,
				display_name,
				photo_url) 
			VALUES (?, ?, ?, ?, ?)
		`
		result, err := sqlClient.Exec(query, user.Email, "google", user.Email, user.DisplayName, user.Picture)

		fmt.Println(err)
		if err != nil {

			var message string
			message = "Duplicated entry"
			http.Error(w, message, http.StatusConflict)
			return
		}
		json.NewEncoder(w).Encode(result)
		// fmt.Fprintf(w, "Hello, %s! Your email is %s. your name is %s, your phoneNumber is %s, your dateOfBirth is %s, your displayName is %s, your localId is %s, your verified is %t", user.Name, user.Email, user.PhoneNumber, user.DateOfBirth, user.DisplayName, user.LocalId, user.Verified)
	}
}
