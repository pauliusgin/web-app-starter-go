package password

import (
	"errors"
	"strings"
	"testing"
)

const plaintext = "correct horse battery staple"

func hashForTest(t *testing.T, plainPassword string) string {
	t.Helper()

	encodedHash, err := Hash(plainPassword)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	return encodedHash
}

func TestVerifyAcceptsTheHashedPassword(t *testing.T) {
	encodedHash := hashForTest(t, plaintext)

	if !strings.HasPrefix(encodedHash, "$argon2id$") {
		t.Errorf("encoded hash = %q, want the $argon2id$ prefix", encodedHash)
	}

	matches, err := Verify(VerificationParams{
		Password: plaintext,
		Hash:     encodedHash,
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !matches {
		t.Error("Verify = false, want true for the password that was hashed")
	}
}

func TestVerifyReportsAWrongPasswordWithoutAnError(t *testing.T) {
	encodedHash := hashForTest(t, plaintext)

	matches, err := Verify(VerificationParams{
		Password: "incorrect horse battery staple",
		Hash:     encodedHash,
	})

	if err != nil {
		t.Fatalf("Verify = %v, want a nil error: a wrong password is not a failure", err)
	}
	if matches {
		t.Error("Verify = true, want false")
	}
}

func TestHashSaltsEveryHashSeparately(t *testing.T) {
	firstHash := hashForTest(t, plaintext)
	secondHash := hashForTest(t, plaintext)

	if firstHash == secondHash {
		t.Error("hashing the same password twice produced identical encoded hashes, so the salt is not random")
	}

	matches, err := Verify(VerificationParams{
		Password: plaintext,
		Hash:     secondHash,
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !matches {
		t.Error("Verify = false, want true against the second hash of the same password")
	}
}

func TestVerifyRejectsAnUnreadableEncodedHash(t *testing.T) {
	cases := []struct {
		name        string
		encodedHash string
	}{
		{name: "empty", encodedHash: ""},
		{name: "garbage", encodedHash: "not-a-hash"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			matches, err := Verify(VerificationParams{
				Password: plaintext,
				Hash:     testCase.encodedHash,
			})
			if matches {
				t.Error("Verify = true, want false for an unreadable encoded hash")
			}

			unreadable := ErrEncodedHashUnreadable{}
			if !errors.As(err, &unreadable) {
				t.Fatalf("error = %v, want ErrEncodedHashUnreadable", err)
			}
			if unreadable.Cause == nil {
				t.Error("Cause = nil, want the argon2id parser error")
			}
		})
	}
}
