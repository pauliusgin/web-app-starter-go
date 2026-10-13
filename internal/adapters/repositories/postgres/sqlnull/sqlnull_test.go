package sqlnull

import (
	"database/sql"
	"testing"
	"time"
)

func TestStringPointerAnswersNilForNull(t *testing.T) {
	if got := StringPointer(sql.NullString{}); got != nil {
		t.Errorf("StringPointer(NULL) = %q, want nil", *got)
	}
}

func TestStringPointerPointsAtAValidColumn(t *testing.T) {
	got := StringPointer(sql.NullString{String: "Ada", Valid: true})
	if got == nil || *got != "Ada" {
		t.Errorf("StringPointer(\"Ada\") = %v, want a pointer to \"Ada\"", got)
	}
}

func TestStringPointerKeepsAValidEmptyStringNonNil(t *testing.T) {
	got := StringPointer(sql.NullString{String: "", Valid: true})
	if got == nil || *got != "" {
		t.Errorf("StringPointer(valid \"\") = %v, want a pointer to \"\"", got)
	}
}

func TestTimePointerAnswersNilForNull(t *testing.T) {
	if got := TimePointer(sql.NullTime{}); got != nil {
		t.Errorf("TimePointer(NULL) = %v, want nil", *got)
	}
}

func TestTimePointerPointsAtAValidColumn(t *testing.T) {
	stamp := time.Date(2026, 3, 15, 8, 30, 0, 0, time.UTC)

	got := TimePointer(sql.NullTime{Time: stamp, Valid: true})
	if got == nil || !got.Equal(stamp) {
		t.Errorf("TimePointer(%v) = %v, want a pointer to it", stamp, got)
	}
}

func TestStringUnlessEmptyStoresTheEmptyStringAsNull(t *testing.T) {
	if got := StringUnlessEmpty(""); got.Valid {
		t.Errorf("StringUnlessEmpty(\"\") = %+v, want NULL", got)
	}
}

func TestStringUnlessEmptyKeepsAnyOtherString(t *testing.T) {
	got := StringUnlessEmpty("ref-42")
	if !got.Valid || got.String != "ref-42" {
		t.Errorf("StringUnlessEmpty(\"ref-42\") = %+v, want valid \"ref-42\"", got)
	}
}

func TestInt32UnlessZeroStoresZeroAsNull(t *testing.T) {
	if got := Int32UnlessZero(0); got.Valid {
		t.Errorf("Int32UnlessZero(0) = %+v, want NULL", got)
	}
}

func TestInt32UnlessZeroKeepsAnyOtherInt(t *testing.T) {
	got := Int32UnlessZero(800)
	if !got.Valid || got.Int32 != 800 {
		t.Errorf("Int32UnlessZero(800) = %+v, want valid 800", got)
	}
}
