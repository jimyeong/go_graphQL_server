package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"example.com/m/v2/helper"
	"github.com/graphql-go/graphql"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
)

var queryType = graphql.NewObject(graphql.ObjectConfig{})

var schema, _ = graphql.NewSchema(graphql.SchemaConfig{})

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

<<<<<<< Updated upstream
	// oauth config
	conf := configOauth(OAUTH_KEY, OAUTH_SECRET, OAUTH_REDIRECT_URL)
=======
	conf := configOauth(OAUTH_KEY, OAUTH_SECRET, OAUTH_REDIRECT_URL, ORIGIN)
>>>>>>> Stashed changes

	// csrf protection token generation
	randomStr := helper.RandomString(10)
	verifier := oauth2.GenerateVerifier()

	// router
	http.HandleFunc("/oauth/google", GoogleOauthLogin(conf, randomStr, verifier))
	http.HandleFunc(OAUTH_REDIRECT_URL, GoogleOauthCallback(conf, randomStr, verifier))

	// start server
	log.Fatal(http.ListenAndServe(":3000", nil))

}

func GoogleOauthLogin(conf *oauth2.Config, randomStr string, verifier string) func(w http.ResponseWriter, r *http.Request) {
	url := conf.AuthCodeURL(randomStr, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	}
}
func GoogleOauthCallback(conf *oauth2.Config, randomStr string, verifier string) func(w http.ResponseWriter, r *http.Request) {
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
		fmt.Println(token)
		userInfo, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
		if err != nil {
			http.Error(w, "Failed to get user info", http.StatusInternalServerError)
			return
		}
		defer userInfo.Body.Close()
		var user struct {
			ID    string `json:"id"`
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		if err := json.NewDecoder(userInfo.Body).Decode(&user); err != nil {
			http.Error(w, "Failed to decode user info", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "Hello, %s! Your email is %s.", user.Name, user.Email)
	}
}
