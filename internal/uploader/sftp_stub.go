//go:build nosftp

package uploader

import "fmt"

// This stub is compiled only with the `nosftp` build tag (used to build/test in
// environments without golang.org/x/crypto). Normal builds use sftp.go.
func newSFTP(_ Destination) (Uploader, error) {
	return nil, fmt.Errorf("this build was compiled without sftp support")
}
