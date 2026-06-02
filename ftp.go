package uploader

import (
	"crypto/tls"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
)

type ftpUploader struct {
	d   Destination
	tls bool
}

func newFTP(d Destination, useTLS bool) (Uploader, error) {
	if d.Host == "" {
		return nil, fmt.Errorf("ftp destination requires \"host\"")
	}
	if d.ShareURL == "" {
		return nil, fmt.Errorf("ftp destination requires \"share_url\" to build the link")
	}
	return &ftpUploader{d: d, tls: useTLS}, nil
}

func (f *ftpUploader) Kind() string {
	if f.tls {
		return "ftps"
	}
	return "ftp"
}

func (f *ftpUploader) Upload(name string, r io.Reader, _ int64) (string, error) {
	port := f.d.Port
	if port == 0 {
		port = 21
	}
	addr := fmt.Sprintf("%s:%d", f.d.Host, port)

	opts := []ftp.DialOption{ftp.DialWithTimeout(30 * time.Second)}
	if f.tls {
		// Explicit FTPS (AUTH TLS), matching the original lftp ssl-auth TLS.
		opts = append(opts, ftp.DialWithExplicitTLS(&tls.Config{
			InsecureSkipVerify: !verify(f.d), // original used verify-certificate no
			ServerName:         f.d.Host,
		}))
	}

	conn, err := ftp.Dial(addr, opts...)
	if err != nil {
		return "", fmt.Errorf("dial: %w", err)
	}
	defer conn.Quit()

	if err := conn.Login(f.d.User, f.d.Pass); err != nil {
		return "", fmt.Errorf("login: %w", err)
	}

	remoteDir := f.d.RemoteDir
	if remoteDir == "" {
		remoteDir = "/"
	}
	remotePath := path.Join(remoteDir, name)

	if err := conn.Stor(remotePath, r); err != nil {
		return "", fmt.Errorf("store: %w", err)
	}

	return strings.TrimRight(f.d.ShareURL, "/") + "/" + name, nil
}
