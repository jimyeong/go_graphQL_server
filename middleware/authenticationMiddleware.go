package middleware

import (
	"fmt"
	"net/http"

	"example.com/m/v2/dbs"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

func AuthenticationMiddleware(store *sessions.CookieStore, secret string, appRedis *dbs.AppRedis) gin.HandlerFunc {
	return func(context *gin.Context) {
		cookies := context.Request.Cookies()
		var session_cookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == "session_id" {
				session_cookie = cookie
			}
			fmt.Println("cookie: ", cookie)
		}

		// when client doesn't have session_id cookie, induce them to login again
		if session_cookie.Value == "" {
			context.Redirect(http.StatusTemporaryRedirect, "/oauth/google")
		}

		// retreive JWT token from redis
		sessions, error := store.Get(context.Request, session_cookie.Value)
		if error != nil {
			context.Redirect(http.StatusTemporaryRedirect, "/oauth/google")
		}
		fmt.Println("sessions: ", sessions)

		issued_token, err := appRedis.GetSessionToken(context.Request.Context(), session_cookie.Value)
		if err != nil {
			/*TODO: logging*/
			// if no token in redis, renew it
			context.Redirect(http.StatusTemporaryRedirect, "/oauth/google")
		}
		// session.Values["jwt_token"] = issued_token
		// session.Save(context.Request, context.Writer)
		context.Set("jwt_token", issued_token)
		context.Next()
	}

}
