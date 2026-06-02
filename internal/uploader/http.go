package uploader

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// httpUploader implements a ShareX-style custom uploader: multipart POST, then
// extract the resulting URL from the response via a JSON key path or a regex.
type httpUploader struct {
	d     Destination
	field string
	re    *regexp.Regexp
	cli   *http.Client
}

func newHTTP(d Destination) (Uploader, error) {
	if d.URL == "" {
		return nil, fmt.Errorf("http destination requires \"url\"")
	}
	field := d.FileField
	if field == "" {
		field = "file"
	}
	var re *regexp.Regexp
	if d.URLRegex != "" {
		var err error
		re, err = regexp.Compile(d.URLRegex)
		if err != nil {
			return nil, fmt.Errorf("bad url_regex: %w", err)
		}
	}
	tr := &http.Transport{}
	if !verify(d) {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &httpUploader{
		d:     d,
		field: field,
		re:    re,
		cli:   &http.Client{Timeout: 120 * time.Second, Transport: tr},
	}, nil
}

func (h *httpUploader) Kind() string { return "http" }

func (h *httpUploader) Upload(name string, r io.Reader, _ int64) (string, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	for k, v := range h.d.FormData {
		if err := mw.WriteField(k, v); err != nil {
			return "", err
		}
	}
	fw, err := mw.CreateFormFile(h.field, name)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(fw, r); err != nil {
		return "", err
	}
	if err := mw.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, h.d.URL, &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	for k, v := range h.d.Headers {
		req.Header.Set(k, v)
	}

	resp, err := h.cli.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("server returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	return h.extractURL(body)
}

// extractURL resolves the share link from the response per the configured rule.
func (h *httpUploader) extractURL(body []byte) (string, error) {
	switch {
	case h.d.URLJSONKey != "":
		var data any
		if err := json.Unmarshal(body, &data); err != nil {
			return "", fmt.Errorf("response not JSON: %w", err)
		}
		v, ok := digJSON(data, strings.Split(h.d.URLJSONKey, "."))
		if !ok {
			return "", fmt.Errorf("url_json_key %q not found in response", h.d.URLJSONKey)
		}
		s, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("url_json_key %q is not a string", h.d.URLJSONKey)
		}
		return s, nil
	case h.re != nil:
		m := h.re.FindSubmatch(body)
		if len(m) < 2 {
			return "", fmt.Errorf("url_regex matched no capture group")
		}
		return string(m[1]), nil
	default:
		// No rule: assume the whole (trimmed) body is the URL, like many simple hosts.
		return strings.TrimSpace(string(body)), nil
	}
}

// digJSON walks a dotted path through decoded JSON (objects only).
func digJSON(v any, path []string) (any, bool) {
	cur := v
	for _, p := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = m[p]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}
