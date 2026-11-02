package config

import "slices"

// AppEnv is the environment the server runs in; only the consts below are valid.
type AppEnv string

const (
	// AppEnvLocal is a developer machine.
	AppEnvLocal AppEnv = "local"
	// AppEnvDev is the shared development deployment.
	AppEnvDev AppEnv = "dev"
	// AppEnvProd is the production deployment.
	AppEnvProd AppEnv = "prod"
)

var knownAppEnvs = []AppEnv{AppEnvLocal, AppEnvDev, AppEnvProd}

func readAppEnvVariable() (AppEnv, error) {
	rawValue, err := readRequiredVariable(readRequiredVariableParams{
		Variable:    AppEnvVariable,
		Expectation: appEnvExpectationText,
	})
	if err != nil {
		return "", err
	}

	appEnv := AppEnv(rawValue)
	if !slices.Contains(knownAppEnvs, appEnv) {
		return "", ErrAppEnvInvalid{Variable: AppEnvVariable, RawValue: rawValue}
	}

	return appEnv, nil
}
