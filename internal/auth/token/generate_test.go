package token

import (
	"encoding/base64"
	"testing"
)

func TestGenerateReturnsAToken(t *testing.T) {
	generated := Generate()

	secret, err := base64.RawURLEncoding.DecodeString(generated.Plaintext)
	if err != nil {
		t.Fatalf("plaintext %q is not raw URL base64: %v", generated.Plaintext, err)
	}
	if len(secret) != tokenByteLength {
		t.Errorf("decoded plaintext = %d bytes, want %d", len(secret), tokenByteLength)
	}
	if generated.Hash != Hash(generated.Plaintext) {
		t.Error("Hash field does not match Hash(Plaintext), so the stored hash would never match the presented token")
	}
	if generated.Plaintext == generated.Hash {
		t.Error("Plaintext equals Hash, so the stored form is the secret itself")
	}
}

func TestGeneratedFromSecretIsDeterministicForItsSecret(t *testing.T) {
	secret := [tokenByteLength]byte{}
	for index := range secret {
		secret[index] = byte(index)
	}

	generated := generatedFromSecret(secret)

	wantPlaintext := "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8"
	wantHash := "ea866a757e4c38babfa8127cbe9a409d3e1f93a00ff1488ff735fcf917afffd0"

	if generated.Plaintext != wantPlaintext {
		t.Errorf("Plaintext = %q, want %q", generated.Plaintext, wantPlaintext)
	}
	if generated.Hash != wantHash {
		t.Errorf("Hash = %q, want %q", generated.Hash, wantHash)
	}
}
