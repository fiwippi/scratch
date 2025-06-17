package halo

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"net/url"
	"strings"

	"github.com/disintegration/imageorient"
	"github.com/fiwippi/halo/internal/api"
	"github.com/fiwippi/halo/internal/stringutil"
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
	mux.HandleFunc("POST   /filter", fc.AddFilter)
	mux.HandleFunc("DELETE /filter", fc.DeleteFilter)
	mux.HandleFunc("GET    /tags", fc.GetTags)
	mux.HandleFunc("POST   /tags", fc.AddTag)
	mux.HandleFunc("DELETE /tags", fc.DeleteTag)
	mux.HandleFunc("PATCH  /tags/{name}", fc.RenameTag)
	mux.HandleFunc("POST   /images", fc.UploadImage)
	mux.HandleFunc("DELETE /images/{id}", fc.DeleteImage)
	mux.HandleFunc("GET    /images/{id}/dialog", fc.GetImageDialog)
	mux.HandleFunc("GET    /images/{id}/tags", fc.GetImageTagsText)
	mux.HandleFunc("POST   /images/{id}/tags", fc.AddTagToImage)
	mux.HandleFunc("DELETE /images/{id}/tags", fc.DeleteTagFromImage)
	mux.HandleFunc("GET    /images/{id}/tags/associated", fc.GetImageTagsAssociatedSelect)
	mux.HandleFunc("GET    /images/{id}/tags/unassociated", fc.GetImageTagsUnassociatedSelect)
	return mux
}

func (fc *fragmentController) AddFilter(w http.ResponseWriter, r *http.Request) {
	newFilter := r.FormValue("filter")
	if newFilter == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filters := stringutil.Deduplicate(append(getFilters(r), newFilter)...)

	w.Header().Set("HX-Redirect", "/?"+url.Values{"tag": filters}.Encode())
	w.WriteHeader(http.StatusOK)
}

func (fc *fragmentController) DeleteFilter(w http.ResponseWriter, r *http.Request) {
	filter := r.FormValue("filter")
	if filter == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filters := getFilters(r)
	for i, f := range filters {
		if filter == f {
			filters = append(filters[:i], filters[i+1:]...)
			break
		}
	}

	w.Header().Set("HX-Redirect", "/?"+url.Values{"tag": filters}.Encode())
	w.WriteHeader(http.StatusOK)
}

func (fc *fragmentController) GetTags(w http.ResponseWriter, r *http.Request) {
	tags, err := fc.store.GetTags()
	if err != nil {
		api.Error(w, fmt.Errorf("get tags: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "tags-select.html", tags); err != nil {
		api.Error(w, fmt.Errorf("exec fragment: %w", err))
	}
}

func (fc *fragmentController) AddTag(w http.ResponseWriter, r *http.Request) {
	newTag := r.Header.Get("HX-Prompt")
	if newTag == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := fc.store.AddTag(newTag); err != nil {
		api.Error(w, fmt.Errorf("add tag: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "filtersUpdated")
	w.WriteHeader(http.StatusOK)
}

func (fc *fragmentController) RenameTag(w http.ResponseWriter, r *http.Request) {
	oldName := r.PathValue("name")
	if oldName == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	newName := r.Header.Get("HX-Prompt")
	if newName == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := fc.store.RenameTag(oldName, newName); err != nil {
		api.Error(w, fmt.Errorf("rename tag: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	filters := getFilters(r)
	for i, f := range filters {
		if oldName == f {
			filters = append(filters[:i], filters[i+1:]...)
			break
		}
	}
	filters = append(filters, newName)

	path := "/?" + url.Values{"tag": filters}.Encode()
	w.Header().Set("HX-Redirect", path)
	w.WriteHeader(http.StatusOK)
}

func (fc *fragmentController) DeleteTag(w http.ResponseWriter, r *http.Request) {
	tag := r.Header.Get("HX-Prompt")
	if tag == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := fc.store.DeleteTag(tag); err != nil {
		api.Error(w, fmt.Errorf("delete tag: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Lots of cleanup and reloading to do if the deleted tag
	// is currently used as a filter, it's simpler to just
	// engage a redirect for the page and ensure the tag is
	// not part of the filters
	filters := getFilters(r)
	for i, f := range filters {
		if tag == f {
			filters = append(filters[:i], filters[i+1:]...)
			break
		}
	}
	path := "/?" + url.Values{"tag": filters}.Encode()
	if path == "/?" {
		path = "/"
	}

	w.Header().Set("HX-Redirect", path)
	w.WriteHeader(http.StatusOK)
}

// curl http://localhost:9090/images -s -X POST -F "upload=@cat.jpeg"
// sqlite3 ./data/store.db "SELECT writefile('store-cat.jpeg',data) FROM originals ORDER BY id DESC LIMIT 1;"
func (fc *fragmentController) UploadImage(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(15 << 20) // Only first 15 MB are buffered in memory
	if err != nil {
		api.Error(w, fmt.Errorf("parse form: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	files := r.MultipartForm.File["upload"]
	if len(files) == 0 {
		api.Error(w, fmt.Errorf("upload empty"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if len(files) > 10 {
		api.Error(w, fmt.Errorf(">10 images specified"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	images := make([]image.Image, len(files))
	for i, header := range files {
		file, err := header.Open()
		if err != nil {
			api.Error(w, fmt.Errorf("open upload %d: %w", i+1, err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer file.Close()

		img, _, err := imageorient.Decode(file) // Default image package messes up orientation sometimes
		if err != nil {
			api.Error(w, fmt.Errorf("decode image %d: %w", i+1, err))
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}
		images[i] = img
	}

	ids, err := fc.store.AddImages(images, r.Form["tag"]...)
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

func (fc fragmentController) GetImageTagsText(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("id")
	id, err := ulid.Parse(idString)
	if err != nil {
		api.Error(w, fmt.Errorf("invalid id: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tags, err := fc.store.GetAssociatedImageTags(id)
	if err != nil {
		api.Error(w, fmt.Errorf("get associated image tags: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err := w.Write([]byte(strings.Join(tags, ", "))); err != nil {
		api.Error(w, fmt.Errorf("write image tags: %w", err))
	}
}

func (fc fragmentController) AddTagToImage(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("id")
	id, err := ulid.Parse(idString)
	if err != nil {
		api.Error(w, fmt.Errorf("invalid id: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	newTag := r.FormValue("tag")
	if newTag == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = fc.store.AddTagToImage(newTag, id)
	if err != nil {
		api.Error(w, fmt.Errorf("add tag to image: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Trigger", "imageTagsUpdated")
	w.WriteHeader(http.StatusOK)
}

func (fc fragmentController) DeleteTagFromImage(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("id")
	id, err := ulid.Parse(idString)
	if err != nil {
		api.Error(w, fmt.Errorf("invalid id: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	tag := r.FormValue("tag")
	if tag == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = fc.store.DeleteTagFromImage(tag, id)
	if err != nil {
		api.Error(w, fmt.Errorf("delete tag from image: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	found := false
	for _, f := range getFilters(r) {
		if tag == f {
			found = true
			break
		}
	}
	if found {
		// If we're deleting a tag that's currently selected,
		// then we need to delete the img tag and dialog
		w.Header().Set("HX-Reswap", "delete")
		w.Header().Set("HX-Retarget", fmt.Sprintf("#img-%s", id))
	} else {
		// If we're not deleting the image, let's refresh its tags
		w.Header().Set("HX-Trigger", "imageTagsUpdated")
	}

	w.WriteHeader(http.StatusOK)
}

func (fc fragmentController) GetImageTagsAssociatedSelect(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("id")
	id, err := ulid.Parse(idString)
	if err != nil {
		api.Error(w, fmt.Errorf("invalid id: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tags, err := fc.store.GetAssociatedImageTags(id)
	if err != nil {
		api.Error(w, fmt.Errorf("get associated image tags: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = tmpl.ExecuteTemplate(w, "image-tags-associated.html", map[string]any{
		"ID":   id.String(),
		"Tags": tags,
	})
	if err != nil {
		api.Error(w, fmt.Errorf("exec fragment: %w", err))
	}
}

func (fc fragmentController) GetImageTagsUnassociatedSelect(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("id")
	id, err := ulid.Parse(idString)
	if err != nil {
		api.Error(w, fmt.Errorf("invalid id: %w", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tags, err := fc.store.GetUnassociatedImageTags(id)
	if err != nil {
		api.Error(w, fmt.Errorf("get unassociated image tags: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = tmpl.ExecuteTemplate(w, "image-tags-unassociated.html", map[string]any{
		"ID":   id.String(),
		"Tags": tags,
	})
	if err != nil {
		api.Error(w, fmt.Errorf("exec fragment: %w", err))
	}
}
