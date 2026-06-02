package uploader

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// s3Uploader performs an S3 PutObject using AWS Signature Version 4 implemented
// against the standard library only. This works with AWS S3 and any
// S3-compatible service (Cloudflare R2, Backblaze B2, Wasabi, MinIO) by setting
// the endpoint and region appropriately.
type s3Uploader struct {
	d        Destination
	endpoint *url.URL
	region   string
	cli      *http.Client
}

func newS3(d Destination) (Uploader, error) {
	if d.Bucket == "" || d.AccessKey == "" || d.SecretKey == "" {
		return nil, fmt.Errorf("s3 destination requires bucket, access_key and secret_key")
	}
	region := d.Region
	if region == "" {
		region = "us-east-1"
	}
	ep := d.Endpoint
	if ep == "" {
		ep = "https://s3." + region + ".amazonaws.com"
	}
	u, err := url.Parse(ep)
	if err != nil {
		return nil, fmt.Errorf("bad endpoint: %w", err)
	}
	tr := &http.Transport{}
	if !verify(d) {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &s3Uploader{
		d:        d,
		endpoint: u,
		region:   region,
		cli:      &http.Client{Timeout: 180 * time.Second, Transport: tr},
	}, nil
}

func (s *s3Uploader) Kind() string { return "s3" }

func (s *s3Uploader) Upload(name string, r io.Reader, size int64) (string, error) {
	key := s.d.KeyPrefix + name

	// Path-style URL: <endpoint>/<bucket>/<key>. Path-style is the most
	// portable across S3-compatible providers.
	reqURL := *s.endpoint
	reqURL.Path = "/" + s.d.Bucket + "/" + key

	body, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	payloadHash := sha256hex(body)

	req, err := http.NewRequest(http.MethodPut, reqURL.String(), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.ContentLength = int64(len(body))
	req.Header.Set("Content-Type", contentTypeFor(name))

	if err := s.sign(req, payloadHash); err != nil {
		return "", err
	}

	resp, err := s.cli.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("s3 returned %s: %s", resp.Status, strings.TrimSpace(string(rb)))
	}

	if s.d.ShareURL != "" {
		return strings.TrimRight(s.d.ShareURL, "/") + "/" + key, nil
	}
	return reqURL.String(), nil
}

// sign adds the SigV4 Authorization header for an S3 request.
func (s *s3Uploader) sign(req *http.Request, payloadHash string) error {
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")

	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	req.Header.Set("Host", req.URL.Host)

	// --- canonical request ---
	signedHeaders, canonicalHeaders := canonHeaders(req)
	canonicalURI := uriEncodePath(req.URL.Path)
	canonicalQuery := req.URL.RawQuery // empty for our PUT
	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI,
		canonicalQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	// --- string to sign ---
	scope := strings.Join([]string{dateStamp, s.region, "s3", "aws4_request"}, "/")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		sha256hex([]byte(canonicalRequest)),
	}, "\n")

	// --- signing key + signature ---
	kDate := hmacSHA256([]byte("AWS4"+s.d.SecretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(s.region))
	kService := hmacSHA256(kRegion, []byte("s3"))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	signature := hex.EncodeToString(hmacSHA256(kSigning, []byte(stringToSign)))

	auth := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		s.d.AccessKey, scope, signedHeaders, signature,
	)
	req.Header.Set("Authorization", auth)
	return nil
}

func canonHeaders(req *http.Request) (signed, canonical string) {
	// Always sign host, x-amz-content-sha256, x-amz-date (+ content-type if set).
	names := []string{"host", "x-amz-content-sha256", "x-amz-date"}
	if req.Header.Get("Content-Type") != "" {
		names = append(names, "content-type")
	}
	sort.Strings(names)

	var b strings.Builder
	for _, n := range names {
		var v string
		switch n {
		case "host":
			v = req.URL.Host
		default:
			v = req.Header.Get(n)
		}
		b.WriteString(n)
		b.WriteString(":")
		b.WriteString(strings.TrimSpace(v))
		b.WriteString("\n")
	}
	return strings.Join(names, ";"), b.String()
}

// uriEncodePath encodes each path segment per AWS rules (slashes preserved).
func uriEncodePath(p string) string {
	if p == "" {
		return "/"
	}
	segs := strings.Split(p, "/")
	for i, s := range segs {
		segs[i] = awsURIEncode(s, false)
	}
	return strings.Join(segs, "/")
}

// awsURIEncode mirrors AWS's required percent-encoding.
func awsURIEncode(s string, encodeSlash bool) string {
	const unreserved = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.~"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case strings.IndexByte(unreserved, c) >= 0:
			b.WriteByte(c)
		case c == '/' && !encodeSlash:
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

func sha256hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func contentTypeFor(name string) string {
	switch {
	case strings.HasSuffix(name, ".png"):
		return "image/png"
	case strings.HasSuffix(name, ".jpg"), strings.HasSuffix(name, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(name, ".gif"):
		return "image/gif"
	case strings.HasSuffix(name, ".webp"):
		return "image/webp"
<<<<<<< HEAD
	case strings.HasSuffix(name, ".mp4"):
		return "video/mp4"
	case strings.HasSuffix(name, ".mov"):
		return "video/quicktime"
	case strings.HasSuffix(name, ".webm"):
		return "video/webm"
	case strings.HasSuffix(name, ".mkv"):
		return "video/x-matroska"
=======
>>>>>>> 2e8b1664db4249d004dea793b3e13c8d8f22bd19
	default:
		return "application/octet-stream"
	}
}
