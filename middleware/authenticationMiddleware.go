package middleware

import (
	"net/http"

	"example.com/m/v2/dbs"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

func AuthenticationMiddleware(store *sessions.CookieStore, secret string, appRedis *dbs.AppRedis) gin.HandlerFunc {
	return func(context *gin.Context) {
		session, _ := store.Get(context.Request, "app_token")
		if session.Values["token"] == nil {
			context.Redirect(http.StatusTemporaryRedirect, "/oauth/google")
		}

		issued_token, err := appRedis.GetAppToken(context.Request.Context(), session.Values["app_token"].(string))
		if err != nil {
			/*TODO: logging*/
			// if no token in redis, renew it
			context.Redirect(http.StatusTemporaryRedirect, "/oauth/google")
		}
		session.Values["jwt_token"] = issued_token
		session.Save(context.Request, context.Writer)
		context.Next()

	}

}
