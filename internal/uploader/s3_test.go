package uploader

import (
	"encoding/hex"
	"testing"
)

// Verify the signing-key derivation against AWS's documented example
// (from "Examples of how to derive a signing key for Signature Version 4").
// Expected final hex from AWS docs:
// c4afb1cc5771d871763a393e44b703571b55cc28424d1a5e86da6ed3c154a4b9
func TestDeriveSigningKey(t *testing.T) {
	secret := "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"
	dateStamp := "20150830"
	region := "us-east-1"
	service := "iam"

	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))

	got := hex.EncodeToString(kSigning)
	want := "c4afb1cc5771d871763a393e44b703571b55cc28424d1a5e86da6ed3c154a4b9"
	if got != want {
		t.Fatalf("signing key mismatch:\n got=%s\nwant=%s", got, want)
	}
}

func TestSHA256Hex(t *testing.T) {
	// sha256("") well-known value.
	got := sha256hex([]byte(""))
	want := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Fatalf("sha256 empty mismatch: got=%s want=%s", got, want)
	}
}

func TestURIEncodePath(t *testing.T) {
	cases := map[string]string{
		"/bucket/my file.png": "/bucket/my%20file.png",
		"/bucket/a~b-c_d.png": "/bucket/a~b-c_d.png",
		"/":                   "/",
	}
	for in, want := range cases {
		if got := uriEncodePath(in); got != want {
			t.Errorf("uriEncodePath(%q)=%q want %q", in, got, want)
		}
	}
}
