package halo

import (
	"embed"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"strconv"

	"github.com/fiwippi/halo/internal/api"
	"github.com/oklog/ulid/v2"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

type staticController struct {
	store *store
}

func newStaticController(s *store) *http.ServeMux {
	sc := &staticController{store: s}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", sc.GetHome)
	mux.HandleFunc("GET /favicon.ico", sc.GetFavicon)
	mux.HandleFunc("GET /assets/{name}", sc.GetAsset)
	mux.HandleFunc("GET /images/{id}", sc.GetImage)
	return mux
}

func (sc *staticController) GetHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	tags := getFilters(r)
	images, err := sc.store.GetImageIDs(tags...)
	if err != nil {
		api.Error(w, fmt.Errorf("get images: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = tmpl.ExecuteTemplate(w, "home.html", map[string]any{
		"Tags":   tags,
		"Images": images,
	})
	if err != nil {
		api.Error(w, fmt.Errorf("render page: %w", err))
	}
}

//go:embed assets/*
var assetsFs embed.FS

func (sc *staticController) GetAsset(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	data, err := assetsFs.ReadFile("assets/" + name)
	if err != nil {
		api.Error(w, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, err := w.Write(data); err != nil {
		api.Error(w, fmt.Errorf("write asset: %w", err))
	}
}

//go:embed assets/favicon.ico
var favicon []byte

func (sc *staticController) GetFavicon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/x-icon")
	if _, err := w.Write(favicon); err != nil {
		api.Error(w, fmt.Errorf("get favicon: %w", err))
	}
}

// curl "http://localhost:9090/images/{id}" -s > dl-cat.jpeg
// curl "http://localhost:9090/images/{id}?thumbnail=true" -s > dl-thumbnail-cat.jpeg
func (sc *staticController) GetImage(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("id")
	id, err := ulid.Parse(idString)
	if err != nil {
		api.Error(w, fmt.Errorf("invalid ID"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var data []byte
	if r.URL.Query().Get("thumbnail") == "true" {
		data, err = sc.store.GetThumbnailImageBytes(id)
		if err != nil {
			api.Error(w, fmt.Errorf("get thumbnail: %w", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		data, err = sc.store.GetOriginalImageBytes(id)
		if err != nil {
			api.Error(w, fmt.Errorf("get original: %w", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(data)), 10))
	if _, err := w.Write(data); err != nil {
		api.Error(w, fmt.Errorf("get image: %w", err))
	}
}
