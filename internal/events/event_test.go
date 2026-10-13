package events

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type challengePayload struct {
	Token string `json:"token"`
}

type unencodablePayload struct {
	Waiter chan int
}

func TestEventEncodesThePayloadsItIsGiven(t *testing.T) {
	owner := uuid.NullUUID{UUID: uuid.New(), Valid: true}

	cases := []struct {
		name             string
		params           NewEventParams
		wantData         string
		wantRedactedData json.RawMessage
	}{
		{
			name: "data only leaves redacted data unset",
			params: NewEventParams{
				Label:  testLabel,
				UserID: owner,
				Data:   challengePayload{Token: testChallengeToken},
			},
			wantData:         `{"token":"` + testChallengeToken + `"}`,
			wantRedactedData: nil,
		},
		{
			name: "data and redacted data are each encoded from their own payload",
			params: NewEventParams{
				Label:        confidentialLabel,
				UserID:       owner,
				Data:         challengePayload{Token: testChallengeToken},
				RedactedData: challengePayload{Token: "redacted"},
			},
			wantData:         `{"token":"` + testChallengeToken + `"}`,
			wantRedactedData: json.RawMessage(`{"token":"redacted"}`),
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			dispatched, err := New(testCase.params)
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			if dispatched.ID == uuid.Nil {
				t.Error("ID is the nil uuid, want a generated one")
			}
			if dispatched.Label != testCase.params.Label {
				t.Errorf("Label = %q, want %q", dispatched.Label, testCase.params.Label)
			}
			if dispatched.UserID != owner {
				t.Errorf("UserID = %v, want %v", dispatched.UserID, owner)
			}
			if string(dispatched.Data) != testCase.wantData {
				t.Errorf("Data = %s, want %s", dispatched.Data, testCase.wantData)
			}
			if string(dispatched.RedactedData) != string(testCase.wantRedactedData) ||
				(dispatched.RedactedData == nil) != (testCase.wantRedactedData == nil) {
				t.Errorf("RedactedData = %s, want %s", dispatched.RedactedData, testCase.wantRedactedData)
			}
		})
	}
}

func TestEventReportsAnUnencodablePayload(t *testing.T) {
	cases := []struct {
		name   string
		params NewEventParams
	}{
		{
			name: "data",
			params: NewEventParams{
				Label:        testLabel,
				Data:         unencodablePayload{Waiter: make(chan int)},
				RedactedData: struct{}{},
			},
		},
		{
			name: "redacted data",
			params: NewEventParams{
				Label:        testLabel,
				Data:         struct{}{},
				RedactedData: unencodablePayload{Waiter: make(chan int)},
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			dispatched, err := New(testCase.params)

			dataUnencodable := ErrEventDataUnencodable{}
			if !errors.As(err, &dataUnencodable) {
				t.Fatalf("error = %v, want ErrEventDataUnencodable", err)
			}
			if dataUnencodable.Label != testLabel {
				t.Errorf("Label = %q, want %q", dataUnencodable.Label, testLabel)
			}
			if dataUnencodable.Cause == nil {
				t.Error("Cause is nil, want the json.Marshal failure")
			}
			if dispatched.ID != uuid.Nil || dispatched.Data != nil || dispatched.RedactedData != nil {
				t.Errorf("event = %+v, want the zero value", dispatched)
			}
		})
	}
}
