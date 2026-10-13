package config

import (
	"errors"
	"os"
	"testing"
)

const testIntVariable = "READ_VARS_TEST_INT"

const testPortVariable = "READ_VARS_TEST_PORT"

const testStringVariable = "READ_VARS_TEST_STRING"

const testRequiredVariable = "READ_VARS_TEST_REQUIRED"

const testRequiredExpectation = "expected a test value"

func unsetVariableForTest(t *testing.T, variable string) {
	t.Helper()

	t.Setenv(variable, "")
	if err := os.Unsetenv(variable); err != nil {
		t.Fatalf("unsetting %s: %v", variable, err)
	}
}

func TestReadIntVariableReadsInt(t *testing.T) {
	t.Setenv(testIntVariable, "42")

	value, err := readIntVariable(readIntVariableParams{Variable: testIntVariable, Fallback: 1})
	if err != nil {
		t.Fatalf("readIntVariable: %v", err)
	}

	if value != 42 {
		t.Errorf("value = %d, want %d", value, 42)
	}
}

func TestReadIntVariableRejectsNonInt(t *testing.T) {
	t.Setenv(testIntVariable, "forty-two")

	_, err := readIntVariable(readIntVariableParams{Variable: testIntVariable, Fallback: 1})

	failedParse := ErrFailedIntParse{}
	if !errors.As(err, &failedParse) {
		t.Fatalf("error = %v, want ErrFailedIntParse", err)
	}
	if failedParse.Variable != testIntVariable {
		t.Errorf("Variable = %q, want %q", failedParse.Variable, testIntVariable)
	}
	if failedParse.RawValue != "forty-two" {
		t.Errorf("RawValue = %q, want %q", failedParse.RawValue, "forty-two")
	}
}

func TestReadPortVariableRejectsPortOutOfRange(t *testing.T) {
	t.Setenv(testPortVariable, "100000")

	_, err := readPortVariable(readIntVariableParams{Variable: testPortVariable, Fallback: 8080})

	invalidPort := ErrInvalidPort{}
	if !errors.As(err, &invalidPort) {
		t.Fatalf("error = %v, want ErrInvalidPort", err)
	}
	if invalidPort.Variable != testPortVariable {
		t.Errorf("Variable = %q, want %q", invalidPort.Variable, testPortVariable)
	}
	if invalidPort.RawValue != "100000" {
		t.Errorf("RawValue = %q, want %q", invalidPort.RawValue, "100000")
	}
}

func TestReadStringVariableReadsValue(t *testing.T) {
	t.Setenv(testStringVariable, "present")

	value := readStringVariable(readStringVariableParams{Variable: testStringVariable, Fallback: "fallback"})

	if value != "present" {
		t.Errorf("value = %q, want %q", value, "present")
	}
}

func TestReadStringVariableAppliesFallbackWhenUnset(t *testing.T) {
	unsetVariableForTest(t, testStringVariable)

	value := readStringVariable(readStringVariableParams{Variable: testStringVariable, Fallback: "fallback"})

	if value != "fallback" {
		t.Errorf("value = %q, want %q", value, "fallback")
	}
}

func TestReadRequiredVariableReadsPresentValue(t *testing.T) {
	t.Setenv(testRequiredVariable, "present")

	value, err := readRequiredVariable(readRequiredVariableParams{Variable: testRequiredVariable, Expectation: testRequiredExpectation})
	if err != nil {
		t.Fatalf("readRequiredVariable: %v", err)
	}

	if value != "present" {
		t.Errorf("value = %q, want %q", value, "present")
	}
}

func TestReadRequiredVariableRejectsMissingValue(t *testing.T) {
	unsetVariableForTest(t, testRequiredVariable)

	_, err := readRequiredVariable(readRequiredVariableParams{Variable: testRequiredVariable, Expectation: testRequiredExpectation})

	assertMissingRequiredVariable(t, err)
}

func TestReadRequiredIntVariableReadsPresentValue(t *testing.T) {
	t.Setenv(testRequiredVariable, "7")

	value, err := readRequiredIntVariable(readRequiredVariableParams{Variable: testRequiredVariable, Expectation: testRequiredExpectation})
	if err != nil {
		t.Fatalf("readRequiredIntVariable: %v", err)
	}

	if value != 7 {
		t.Errorf("value = %d, want %d", value, 7)
	}
}

func TestReadRequiredIntVariableRejectsMissingValue(t *testing.T) {
	unsetVariableForTest(t, testRequiredVariable)

	_, err := readRequiredIntVariable(readRequiredVariableParams{Variable: testRequiredVariable, Expectation: testRequiredExpectation})

	assertMissingRequiredVariable(t, err)
}

func assertMissingRequiredVariable(t *testing.T, err error) {
	t.Helper()

	missingVariable := ErrMissingRequiredVariable{}
	if !errors.As(err, &missingVariable) {
		t.Fatalf("error = %v, want ErrMissingRequiredVariable", err)
	}
	if missingVariable.Variable != testRequiredVariable {
		t.Errorf("Variable = %q, want %q", missingVariable.Variable, testRequiredVariable)
	}
	if missingVariable.Expectation != testRequiredExpectation {
		t.Errorf("Expectation = %q, want %q", missingVariable.Expectation, testRequiredExpectation)
	}
}
