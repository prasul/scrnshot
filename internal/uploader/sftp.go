//go:build !nosftp

package uploader

import (
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type sftpUploader struct {
	d Destination
}

func newSFTP(d Destination) (Uploader, error) {
	if d.Host == "" {
		return nil, fmt.Errorf("sftp destination requires \"host\"")
	}
	if d.ShareURL == "" {
		return nil, fmt.Errorf("sftp destination requires \"share_url\" to build the link")
	}
	return &sftpUploader{d: d}, nil
}

func (s *sftpUploader) Kind() string { return "sftp" }

func (s *sftpUploader) Upload(name string, r io.Reader, _ int64) (string, error) {
	port := s.d.Port
	if port == 0 {
		port = 22
	}

	var auths []ssh.AuthMethod
	if s.d.PrivateKey != "" {
		key, err := os.ReadFile(s.d.PrivateKey)
		if err != nil {
			return "", fmt.Errorf("read private key: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return "", fmt.Errorf("parse private key: %w", err)
		}
		auths = append(auths, ssh.PublicKeys(signer))
	}
	if s.d.Pass != "" {
		auths = append(auths, ssh.Password(s.d.Pass))
	}
	if len(auths) == 0 {
		return "", fmt.Errorf("sftp needs either pass or private_key")
	}

	cfg := &ssh.ClientConfig{
		User:            s.d.User,
		Auth:            auths,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // matches original's relaxed verification
		Timeout:         30 * time.Second,
	}

	addr := net.JoinHostPort(s.d.Host, fmt.Sprintf("%d", port))
	conn, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return "", fmt.Errorf("ssh dial: %w", err)
	}
	defer conn.Close()

	client, err := sftp.NewClient(conn)
	if err != nil {
		return "", fmt.Errorf("sftp client: %w", err)
	}
	defer client.Close()

	remoteDir := s.d.RemoteDir
	if remoteDir == "" {
		remoteDir = "/"
	}
	_ = client.MkdirAll(remoteDir)
	remotePath := path.Join(remoteDir, name)

	f, err := client.Create(remotePath)
	if err != nil {
		return "", fmt.Errorf("create remote: %w", err)
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return "", fmt.Errorf("write remote: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", err
	}

	return strings.TrimRight(s.d.ShareURL, "/") + "/" + name, nil
}
