package config

import "errors"

const (
	// FirebaseCredentialsVariable names the env var holding the base64 service account JSON.
	FirebaseCredentialsVariable = "GOOGLE_FIREBASE_CREDENTIALS"
	// FirebaseStorageBucketVariable names the env var holding the bucket uploads are stored in.
	FirebaseStorageBucketVariable = "GOOGLE_FIREBASE_STORAGE_BUCKET"
)

// FirebaseConfig is the Firebase project this service stores uploaded files in.
type FirebaseConfig struct {
	EncodedCredentials string
	StorageBucket      string
}

// EnvVariables is every env variable this config is read from.
func (FirebaseConfig) EnvVariables() []string {
	return []string{
		FirebaseCredentialsVariable,
		FirebaseStorageBucketVariable,
	}
}

func loadFirebaseConfig() (FirebaseConfig, error) {
	encodedCredentials, credentialsErr := readRequiredVariable(readRequiredVariableParams{
		Variable:    FirebaseCredentialsVariable,
		Expectation: firebaseCredentialsExpectationText,
	})

	storageBucket, storageBucketErr := readRequiredVariable(readRequiredVariableParams{
		Variable:    FirebaseStorageBucketVariable,
		Expectation: firebaseStorageBucketExpectationText,
	})

	if err := errors.Join(credentialsErr, storageBucketErr); err != nil {
		return FirebaseConfig{}, err
	}

	return FirebaseConfig{
		EncodedCredentials: encodedCredentials,
		StorageBucket:      storageBucket,
	}, nil
}
