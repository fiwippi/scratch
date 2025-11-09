package halo

import (
	"bytes"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/disintegration/imageorient"
	"github.com/fiwippi/halo/internal/api"
	_ "github.com/jdeng/goheif"
	"github.com/oklog/ulid/v2"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

type fragmentController struct {
	store *store
}

func newFragmentController(s *store) *http.ServeMux {
	fc := &fragmentController{store: s}
	mux := http.NewServeMux()
	mux.HandleFunc("GET    /tags", fc.GetTags)
	mux.HandleFunc("POST   /tags", fc.AddTag)
	mux.HandleFunc("DELETE /tags", fc.DeleteTag)
	mux.HandleFunc("POST   /images", fc.UploadImage)
	mux.HandleFunc("DELETE /images/{id}", fc.DeleteImage)
	mux.HandleFunc("GET    /images/{id}/dialog", fc.GetImageDialog)
	return mux
}

func (fc *fragmentController) GetTags(w http.ResponseWriter, r *http.Request) {
	tags, err := fc.store.GetTags()
	if err != nil {
		api.Error(w, fmt.Errorf("get tags: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "tags-list.html", tags); err != nil {
		api.Error(w, fmt.Errorf("exec fragment: %w", err))
	}
}

func tryQueryUnescape(s string) string {
	unescaped, err := url.QueryUnescape(s)
	if err != nil {
		return s
	}
	return unescaped
}

func (fc *fragmentController) AddTag(w http.ResponseWriter, r *http.Request) {
	newTag := tryQueryUnescape(r.Header.Get("HX-Prompt"))
	if newTag == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := fc.store.AddTag(newTag); err != nil {
		api.Error(w, fmt.Errorf("add tag: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "tagsUpdated")
	w.WriteHeader(http.StatusOK)
}

func (fc *fragmentController) DeleteTag(w http.ResponseWriter, r *http.Request) {
	tag := tryQueryUnescape(r.Header.Get("HX-Prompt"))
	if tag == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := fc.store.DeleteTag(tag); err != nil {
		api.Error(w, fmt.Errorf("delete tag: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

func extractDateTime(file io.Reader) (time.Time, error) {
	cmd := exec.Command("exiftool", "-DateTimeOriginal", "-CreateDate", "-ModifyDate", "-s3", "-")
	cmd.Stdin = file
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return time.Time{}, err
	}

	for line := range strings.SplitSeq(out.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		t, err := time.Parse("2006:01:02 15:04:05", line)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("tag not present")
}

func (fc *fragmentController) UploadImage(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(50 << 20) // Only first 50 MB are buffered in memory
	if err != nil {
		api.Error(w, fmt.Errorf("parse form: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tag := r.Form.Get("tag")
	if tag == "" || !fc.store.HasTag(tag) {
		api.Error(w, fmt.Errorf("invalid tag: %q", tag))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["upload"]
	if len(files) == 0 {
		api.Error(w, fmt.Errorf("upload empty"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if len(files) > 25 {
		api.Error(w, fmt.Errorf(">25 images specified"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	images := make([]newImageData, len(files))
	for i, header := range files {
		file, err := header.Open()
		if err != nil {
			api.Error(w, fmt.Errorf("open upload %d: %w", i+1, err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer file.Close()

		timestamp, err := extractDateTime(file)
		if err != nil {
			api.Error(w, fmt.Errorf("extract date-time %d: %w", i+1, err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// We're performing a second read
		// so we need to rewind the buffer
		file.Seek(0, 0)

		img, _, err := imageorient.Decode(file) // Default image package messes up orientation sometimes
		if err != nil {
			api.Error(w, fmt.Errorf("decode image %d: %w", i+1, err))
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		images[i] = newImageData{
			img:       img,
			timestamp: timestamp,
		}

		// Eagerly close the file to save resources
		// ASAP even though we still guarantee a
		// close with the defer call
		file.Close()
	}

	ids, err := fc.store.AddImages(images, tag)
	if err != nil {
		api.Error(w, fmt.Errorf("add image: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	idsString := make([]string, len(ids))
	for i, id := range ids {
		idsString[i] = id.String()
	}
	w.WriteHeader(http.StatusCreated)
	if err := tmpl.ExecuteTemplate(w, "images.html", idsString); err != nil {
		api.Error(w, fmt.Errorf("exec fragment: %w", err))
	}
}

func (fc *fragmentController) DeleteImage(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("id")
	id, err := ulid.Parse(idString)
	if err != nil {
		api.Error(w, fmt.Errorf("invalid id: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = fc.store.DeleteImage(id)
	if err != nil {
		api.Error(w, fmt.Errorf("delete image image: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (fc *fragmentController) GetImageDialog(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("id")
	id, err := ulid.Parse(idString)
	if err != nil {
		api.Error(w, fmt.Errorf("invalid id: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "image-dialog.html", id.String()); err != nil {
		api.Error(w, fmt.Errorf("exec fragment: %w", err))
	}
}
