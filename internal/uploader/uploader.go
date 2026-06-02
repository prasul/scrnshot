// Package uploader defines a pluggable destination interface in the spirit of
// ShareX's "custom uploaders": a single config file can describe FTP, SFTP,
// S3-compatible, or arbitrary HTTP POST destinations, and the right backend is
// constructed at runtime from the "type" field.
package uploader

import (
	"fmt"
	"io"
)

// Uploader uploads one file and returns the public share URL for it.
//
// name is the (already randomized) remote filename. r streams the file bytes.
// size is the total length, which some backends (S3) require up front.
type Uploader interface {
	Upload(name string, r io.Reader, size int64) (shareURL string, err error)
	// Kind returns a short label for logging, e.g. "ftps" or "s3".
	Kind() string
}

// Destination is the JSON shape of one configured destination. Only the fields
// relevant to the chosen Type need to be filled in. This mirrors ShareX's
// single-object-describes-everything approach so a destination is copy-pasteable.
type Destination struct {
	Type string `json:"type"` // ftp | ftps | sftp | s3 | http

	// Common to FTP/FTPS/SFTP.
	Host       string `json:"host,omitempty"`
	Port       int    `json:"port,omitempty"`
	User       string `json:"user,omitempty"`
	Pass       string `json:"pass,omitempty"`
	RemoteDir  string `json:"remote_dir,omitempty"`  // path on server to drop files, default "/"
	VerifyCert *bool  `json:"verify_cert,omitempty"` // FTPS/HTTPS; default true. The original lftp used false.

	// SFTP-only.
	PrivateKey string `json:"private_key,omitempty"` // path to an OpenSSH private key (optional; falls back to Pass)

	// S3-compatible (AWS, Cloudflare R2, Backblaze B2, Wasabi, MinIO).
	Endpoint  string `json:"endpoint,omitempty"` // e.g. https://<account>.r2.cloudflarestorage.com ; empty = AWS
	Region    string `json:"region,omitempty"`   // e.g. us-east-1 or "auto" for R2
	Bucket    string `json:"bucket,omitempty"`
	AccessKey string `json:"access_key,omitempty"`
	SecretKey string `json:"secret_key,omitempty"`
	KeyPrefix string `json:"key_prefix,omitempty"` // object key prefix, e.g. "screens/"

	// HTTP custom uploader (ShareX-style).
	URL        string            `json:"url,omitempty"`          // POST endpoint
	FileField  string            `json:"file_field,omitempty"`   // multipart field name, default "file"
	Headers    map[string]string `json:"headers,omitempty"`      // extra headers, e.g. Authorization
	FormData   map[string]string `json:"form_data,omitempty"`    // extra multipart text fields
	URLJSONKey string            `json:"url_json_key,omitempty"` // dotted path into JSON response holding the URL
	URLRegex   string            `json:"url_regex,omitempty"`    // alternative: first capture group of this regex

	// Shared.
	// ShareURL is an optional base used to construct the returned link for
	// backends that don't return one themselves (FTP/SFTP/S3). The final URL is
	// ShareURL + "/" + remoteName. For HTTP, the URL comes from the response.
	ShareURL string `json:"share_url,omitempty"`
}

func verify(d Destination) bool {
	if d.VerifyCert == nil {
		return true
	}
	return *d.VerifyCert
}

// New builds the concrete Uploader for a destination.
func New(d Destination) (Uploader, error) {
	switch d.Type {
	case "ftp":
		return newFTP(d, false)
	case "ftps":
		return newFTP(d, true)
	case "s3":
		return newS3(d)
	case "http":
		return newHTTP(d)
	case "sftp":
		return newSFTP(d)
	case "":
		return nil, fmt.Errorf("destination has no \"type\"")
	default:
		return nil, fmt.Errorf("unknown destination type %q", d.Type)
	}
}
