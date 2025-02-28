package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
	"github.com/graphql-go/graphql"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"

	"example.com/m/v2/constants"
	"example.com/m/v2/dbs"
	"example.com/m/v2/helper"
	"example.com/m/v2/middleware"
)

type cookieData struct {
	Email       string `json:"email"`
	SurName     string `json:"surName"`
	FirstName   string `json:"firstName"`
	Picture     string `json:"picture"`
	DisplayName string `json:"displayName"`
	DateOfBirth string `json:"dateOfBirth"`
	Verified    bool   `json:"verified"`
}

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

// token on client side
// 1. saving token in cookie
// 2.storages like Session, Local, Redis,

/**


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

func configSQLDB(context context.Context, DB_HOST string, DB_USER string, DB_PASSWORD string, DB_NAME string) (*sql.DB, error) {

	DB_CONNECTION := fmt.Sprintf("%s:%s@tcp(%s)/%s", DB_USER, DB_PASSWORD, DB_HOST, DB_NAME)
	fmt.Print(DB_CONNECTION)
	mysqlClient, err := sql.Open("mysql", DB_CONNECTION)
	if err != nil {
		panic(err)
	}
	return mysqlClient, err
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
	var REDIS_HOST string = os.Getenv("REDIS_HOST")
	// var REDIS_DB_NAME string = os.Getenv("REDIS_DB_NAME")
	var REDIS_PASSWORD string = os.Getenv("REDIS_PASSWORD")
	var SECRET string = os.Getenv("SECRET")
	conf := configOauth(OAUTH_KEY, OAUTH_SECRET, OAUTH_REDIRECT_URL, ORIGIN)
	mysqlClient, err := configSQLDB(context.Background(), MYSQL_HOST, MYSQL_USER, MYSQL_PASSWORD, MYSQL_DB)
	if err != nil {
		log.Fatal(err)
	}
	defer mysqlClient.Close()
	server := gin.Default()

	_redisDb := dbs.InitRedisDb(context.Background(), REDIS_HOST, REDIS_PASSWORD)

	// csrf protection token generation
	randomStr := helper.RandomString(10)
	verifier := oauth2.GenerateVerifier()

	store := sessions.NewCookieStore([]byte(SECRET))
	server.Use(middleware.AuthenticationMiddleware(store, SECRET, _redisDb))

	// middleware
	// server.Use(tokenCheckMiddleware())
	// router
	server.GET("/oauth/google", GoogleOauthLogin(conf, randomStr, verifier))
	server.GET(OAUTH_REDIRECT_URL, GoogleOauthCallback(conf, randomStr, verifier, mysqlClient, _redisDb, constants.TIME_MINUTES_COOKIE_EXPIRATION))
	// start server
	log.Fatal(server.Run(":3000"))
}
func GoogleOauthLogin(conf *oauth2.Config, randomStr string, verifier string) func(context *gin.Context) {
	url := conf.AuthCodeURL(randomStr, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))
	return func(context *gin.Context) {
		context.Redirect(http.StatusTemporaryRedirect, url)
		//  context.JSON(http.StatusOK, gin.H{"url": url})
	}
}

func GoogleOauthCallback(conf *oauth2.Config, randomStr string, verifier string, sqlClient *sql.DB, redisDb *dbs.AppRedis, TOKEN_EXPIRATION_TIME_MINUTES int) func(context *gin.Context) {
	return func(context *gin.Context) {
		// csrf protection
		if context.Query("state") != randomStr {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid OAuth state"})
			return
		}
		code := context.Query("code")
		token, err := conf.Exchange(context, code, oauth2.VerifierOption(verifier))
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Code exchange failed"})
			return
		}
		// Use token to get user information
		client := conf.Client(context, token)
		userInfo, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
			return
		}
		defer userInfo.Body.Close()
		if err := json.NewDecoder(userInfo.Body).Decode(&user); err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode user info"})
			return
		}

		if user.Email == "" {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Email is required"})
			return
		}

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
		// error cases
		// 1. Duplicated entry => No need to save in DB, but need to save in Redis as we are using 3rd party auth
		// logging the error

		// 2. Other errors
		// fullName := strings.Split(user.Name, " ")

		// userData := cookieData{
		// 	Email: user.Email,

		// 	SurName:     fullName[1],
		// 	FirstName:   fullName[0],
		// 	Picture:     user.Picture,
		// 	DisplayName: user.DisplayName,
		// 	DateOfBirth: user.DateOfBirth,
		// 	Verified:    user.Verified,
		// }
		if err != nil {
			// var message string
			// message = "Duplicated entry"
			// context.JSON(http.StatusConflict, gin.H{"error": message})

			// return
		}
		error := redisDb.SetOauthToken(context, user.Email, token)
		if error != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": error})
			return
		}

		context.JSON(http.StatusOK, gin.H{"result": user.Email, "minutes": constants.TIME_MINUTES_TOKEN_EXPIRATION})
		fmt.Println(result)
		// fmt.Fprintf(w, "Hello, %s! Your email is %s. your name is %s, your phoneNumber is %s, your dateOfBirth is %s, your displayName is %s, your localId is %s, your verified is %t", user.Name, user.Email, user.PhoneNumber, user.DateOfBirth, user.DisplayName, user.LocalId, user.Verified)
	}
}
