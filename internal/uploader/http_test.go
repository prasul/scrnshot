package uploader

import (
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPUploaderJSONKey(t *testing.T) {
	var gotField, gotFilename, gotBody, gotAuth, gotFormVal string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		gotFormVal = r.FormValue("album")
		for field := range r.MultipartForm.File {
			gotField = field
			fh := r.MultipartForm.File[field][0]
			gotFilename = fh.Filename
			f, _ := fh.Open()
			b, _ := io.ReadAll(f)
			gotBody = string(b)
			f.Close()
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"status":"ok","data":{"url":"https://cdn.example.com/abc.png"}}`)
	}))
	defer srv.Close()

	up, err := newHTTP(Destination{
		Type:       "http",
		URL:        srv.URL,
		FileField:  "image",
		Headers:    map[string]string{"Authorization": "Bearer XYZ"},
		FormData:   map[string]string{"album": "screens"},
		URLJSONKey: "data.url",
	})
	if err != nil {
		t.Fatal(err)
	}

	url, err := up.Upload("abc.png", strings.NewReader("PNGDATA"), 7)
	if err != nil {
		t.Fatal(err)
	}

	if url != "https://cdn.example.com/abc.png" {
		t.Errorf("url = %q", url)
	}
	if gotField != "image" {
		t.Errorf("field = %q want image", gotField)
	}
	if gotFilename != "abc.png" {
		t.Errorf("filename = %q", gotFilename)
	}
	if gotBody != "PNGDATA" {
		t.Errorf("body = %q", gotBody)
	}
	if gotAuth != "Bearer XYZ" {
		t.Errorf("auth = %q", gotAuth)
	}
	if gotFormVal != "screens" {
		t.Errorf("form album = %q", gotFormVal)
	}
}

func TestHTTPUploaderRegex(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseMultipartForm(1 << 20)
		io.WriteString(w, "Upload complete. Link: https://i.host/xY9.png done")
	}))
	defer srv.Close()

	up, _ := newHTTP(Destination{
		Type:     "http",
		URL:      srv.URL,
		URLRegex: `(https://\S+\.png)`,
	})
	url, err := up.Upload("x.png", strings.NewReader("data"), 4)
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://i.host/xY9.png" {
		t.Errorf("url = %q", url)
	}
}

var _ = multipart.NewWriter // keep import if test trimmed
