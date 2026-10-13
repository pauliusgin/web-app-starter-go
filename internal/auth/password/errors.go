package password

import "fmt"

const hashingFailedText = "failed hashing the password with argon2id"

const encodedHashUnreadableText = "expected an argon2id encoded hash"

// ErrHashingFailed reports a failed argon2id hash, deliberately carrying no plaintext password.
type ErrHashingFailed struct {
	Cause error
}

func (e ErrHashingFailed) Error() string {
	return fmt.Sprintf("%s: %s", hashingFailedText, e.Cause)
}

func (e ErrHashingFailed) Unwrap() error {
	return e.Cause
}

// ErrEncodedHashUnreadable reports a corrupt stored hash, which is not a wrong password.
type ErrEncodedHashUnreadable struct {
	Cause error
}

func (e ErrEncodedHashUnreadable) Error() string {
	return fmt.Sprintf("unreadable encoded hash: %s: %s", e.Cause, encodedHashUnreadableText)
}

func (e ErrEncodedHashUnreadable) Unwrap() error {
	return e.Cause
}
