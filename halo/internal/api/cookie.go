package api

import (
	"encoding/base64"
	"net/http"
)

func StoreCookie(w http.ResponseWriter, name string, value string) error {
	cookie := &http.Cookie{
		Name:     name,
		Value:    base64.StdEncoding.EncodeToString([]byte(value)),
		Path:     "/",
		MaxAge:   86400, // 1 day
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	return nil
}

func LoadCookie(r *http.Request, name string) (string, error) {
	cookie, _ := r.Cookie(name)
	if cookie == nil {
		return "", nil
	}

	dec, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		return "", err
	}
	return string(dec), nil
}
