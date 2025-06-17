package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
)

func StoreCookieList(w http.ResponseWriter, name string, list []string) error {
	jsonTags, err := json.Marshal(list)
	if err != nil {
		return err
	}
	cookie := &http.Cookie{
		Name:     name,
		Value:    base64.StdEncoding.EncodeToString(jsonTags),
		Path:     "/",
		MaxAge:   86400, // 1 day
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	return nil
}

func LoadCookieList(r *http.Request, name string) ([]string, error) {
	cookie, _ := r.Cookie(name)
	if cookie == nil {
		return []string{}, nil
	}

	dec, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		return []string{}, err
	}
	var tags []string
	if err := json.Unmarshal(dec, &tags); err != nil {
		return []string{}, err
	}

	return tags, nil
}
