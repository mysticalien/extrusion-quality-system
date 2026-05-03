package password

import "testing"

func TestBcryptHasherHashAndCheckSuccess(t *testing.T) {
	hasher := NewBcryptHasher(4)

	hash, err := hasher.Hash("correct-password")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	if hash == "" {
		t.Fatal("expected non-empty hash")
	}

	if hash == "correct-password" {
		t.Fatal("hash must not equal raw password")
	}

	if !hasher.Check("correct-password", hash) {
		t.Fatal("expected password check to succeed")
	}
}

func TestBcryptHasherCheckWrongPassword(t *testing.T) {
	hasher := NewBcryptHasher(4)

	hash, err := hasher.Hash("correct-password")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	if hasher.Check("wrong-password", hash) {
		t.Fatal("expected password check to fail")
	}
}

func TestBcryptHasherCheckInvalidHash(t *testing.T) {
	hasher := NewBcryptHasher(4)

	if hasher.Check("password", "not-a-bcrypt-hash") {
		t.Fatal("expected password check to fail for invalid hash")
	}
}
