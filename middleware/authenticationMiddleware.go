package middleware

import (
	"fmt"
	"net/http"

	"example.com/m/v2/dbs"
	"github.com/gin-gonic/gin"
)

// check for requests coming
func AuthenticationMiddleware(appRedis *dbs.AppRedis) gin.HandlerFunc {
	return func(context *gin.Context) {

		session_id, err := context.Cookie("session_id")
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": "session is not found"})
			return
		}
		issued_token, err := appRedis.GetSessionToken(context.Request.Context(), session_id)
		if err != nil {
			/*TODO: logging*/
			// if no token in redis, renew it
			// context.Redirect(http.StatusTemporaryRedirect, "/oauth/google")
			fmt.Println("error: ", err)
		}
		// session.Values["jwt_token"] = issued_token
		// session.Save(context.Request, context.Writer)
		context.Set("jwt_token", issued_token)
		context.Next()
	}

}
