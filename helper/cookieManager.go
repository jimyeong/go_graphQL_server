package helper

import (
	"net/http"

	"example.com/m/v2/constants"
	"github.com/gorilla/sessions"
)

type appCookieStore struct {
	CookieStore *sessions.CookieStore
}

func initCookieStore(secretKey string) *sessions.CookieStore {
	cookieStore := sessions.NewCookieStore([]byte(secretKey))
	cookieStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   constants.TIME_MINUTES_COOKIE_EXPIRATION * 60,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Domain:   constants.DOMAIN_CLIENT,
	}
	appCookieStore := &appCookieStore{CookieStore: cookieStore}
	return appCookieStore.CookieStore
}
