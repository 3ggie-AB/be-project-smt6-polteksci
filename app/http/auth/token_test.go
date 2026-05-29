package auth

import "testing"

func TestTokenHash(t *testing.T) {
	token := "plain-session-token"
	hash := TokenHash(token)

	if len(hash) != 64 {
		t.Fatalf("expected sha256 hex length 64, got %d", len(hash))
	}
	if hash == token {
		t.Fatal("hash must not equal the original token")
	}
	if hash != TokenHash(token) {
		t.Fatal("hash must be deterministic")
	}
}
