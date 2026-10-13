// Package handlers holds the request decoding and JSON response helpers the HTTP handlers share.
package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
)

const maxRequestBodyBytes int64 = 64 << 10

const jsonMediaType = "application/json"

// DecodeRequest decodes a JSON body holding exactly one T and no unknown fields.
func DecodeRequest[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var zero T

	if err := requireJSONContentType(r); err != nil {
		return zero, err
	}
	if r.Body == nil {
		return zero, ErrJSONRequestBodyUndecodable{Cause: errEmptyRequestBody}
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var request T
	if err := decoder.Decode(&request); err != nil {
		return zero, translateDecodeFailure(err)
	}
	if err := requireSingleJSONValue(decoder); err != nil {
		return zero, err
	}

	return request, nil
}

// RequestDecodingRules answers the failures DecodeRequest reports.
var RequestDecodingRules = []RejectionRule{
	RejectWhen[ErrJSONRequestBodyUndecodable](Rejection{Status: http.StatusBadRequest, Code: ErrorCodeRequestBodyInvalid}),
	RejectWhen[ErrRequestBodyTooLarge](Rejection{Status: http.StatusRequestEntityTooLarge, Code: ErrorCodeRequestBodyTooLarge}),
	RejectWhen[ErrUnsupportedContentType](Rejection{Status: http.StatusUnsupportedMediaType, Code: ErrorCodeUnsupportedContentType}),
}

func requireJSONContentType(r *http.Request) error {
	rawContentType := r.Header.Get("Content-Type")

	mediaType, _, err := mime.ParseMediaType(rawContentType)
	if err != nil || mediaType != jsonMediaType {
		return ErrUnsupportedContentType{ContentType: rawContentType}
	}

	return nil
}

func requireSingleJSONValue(decoder *json.Decoder) error {
	err := decoder.Decode(&json.RawMessage{})

	switch {
	case errors.Is(err, io.EOF):
		return nil
	case err == nil:
		return ErrJSONRequestBodyUndecodable{Cause: errTrailingRequestBody}
	default:
		return translateDecodeFailure(err)
	}
}

func translateDecodeFailure(err error) error {
	if bodyTooLarge, ok := errors.AsType[*http.MaxBytesError](err); ok {
		return ErrRequestBodyTooLarge{Limit: bodyTooLarge.Limit}
	}
	if errors.Is(err, io.EOF) {
		return ErrJSONRequestBodyUndecodable{Cause: errEmptyRequestBody}
	}
	if wrongType, ok := errors.AsType[*json.UnmarshalTypeError](err); ok {
		return ErrJSONRequestBodyUndecodable{Cause: describeFieldTypeMismatch(wrongType)}
	}

	return ErrJSONRequestBodyUndecodable{Cause: err}
}

func describeFieldTypeMismatch(err *json.UnmarshalTypeError) error {
	if err.Field == "" {
		return fmt.Errorf("the JSON %s value does not fit the expected shape", err.Value)
	}

	return fmt.Errorf("field %q holds a JSON %s, which does not fit the expected shape", err.Field, err.Value)
}
