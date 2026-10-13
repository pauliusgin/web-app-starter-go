package config

import (
	"log/slog"
	"os"
	"strconv"
)

const (
	minimumPort = 1
	maximumPort = 65535
)

type readIntVariableParams struct {
	Variable string
	Fallback int
}

func readIntVariable(params readIntVariableParams) (int, error) {
	rawValue := os.Getenv(params.Variable)
	if rawValue == "" {
		logDefaultApplied(params.Variable, params.Fallback)
		return params.Fallback, nil
	}

	intValue, err := strconv.Atoi(rawValue)
	if err != nil {
		return 0, ErrFailedIntParse{Variable: params.Variable, RawValue: rawValue}
	}

	return intValue, nil
}

func readPortVariable(params readIntVariableParams) (int, error) {
	port, err := readIntVariable(params)
	if err != nil {
		return 0, err
	}

	if port < minimumPort || port > maximumPort {
		return 0, ErrInvalidPort{Variable: params.Variable, RawValue: os.Getenv(params.Variable)}
	}

	return port, nil
}

type readStringVariableParams struct {
	Variable string
	Fallback string
}

func readStringVariable(params readStringVariableParams) string {
	rawValue := os.Getenv(params.Variable)
	if rawValue == "" {
		logDefaultApplied(params.Variable, params.Fallback)
		return params.Fallback
	}

	return rawValue
}

type readRequiredVariableParams struct {
	Variable    string
	Expectation string
}

func readRequiredVariable(params readRequiredVariableParams) (string, error) {
	rawValue, isSet := os.LookupEnv(params.Variable)
	if !isSet || rawValue == "" {
		return "", ErrMissingRequiredVariable{Variable: params.Variable, Expectation: params.Expectation}
	}

	return rawValue, nil
}

func readRequiredIntVariable(params readRequiredVariableParams) (int, error) {
	rawValue, err := readRequiredVariable(params)
	if err != nil {
		return 0, err
	}

	intValue, err := strconv.Atoi(rawValue)
	if err != nil {
		return 0, ErrFailedIntParse{Variable: params.Variable, RawValue: rawValue}
	}

	return intValue, nil
}

func logDefaultApplied(variable string, defaultValue any) {
	slog.Info("configuration default applied", "variable", variable, "default", defaultValue)
}
