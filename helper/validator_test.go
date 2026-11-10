package helper

import (
	"strings"
	"testing"
	"time"

	"modul6/model"
)

func TestValidateStructUsesJSONNamesAndCustomRules(t *testing.T) {
	request := model.CreateStudentRequest{NIM: "12a", Name: "A", Grade: 5}
	fields := ValidateStruct(request)
	for _, field := range []string{"nim", "name", "grade"} {
		if _, ok := fields[field]; !ok {
			t.Errorf("expected validation error for %q, got %#v", field, fields)
		}
	}
}

func TestPatchValidationSkipsNilButRejectsEmptyString(t *testing.T) {
	if fields := ValidateStruct(model.PatchStudentRequest{IsActive: new(bool)}); fields != nil {
		t.Fatalf("omitted fields should pass validation, got %#v", fields)
	}
	empty := ""
	fields := ValidateStruct(model.PatchStudentRequest{Name: &empty})
	if fields["name"] == "" {
		t.Fatalf("empty name should have a translated error, got %#v", fields)
	}
}

func TestRegisterPasswordValidation(t *testing.T) {
	tests := []struct {
		password string
		want     string
	}{
		{"password123", "password terlalu umum"},
		{"short1", "minimal 8 karakter"},
		{"longpassword", "harus memuat huruf dan angka"},
	}
	for _, test := range tests {
		request := model.RegisterRequest{
			Username: "tester", Password: test.password, Role: "user",
		}
		fields := ValidateStruct(request)
		if fields["password"] != test.want {
			t.Errorf("password %q: want %q, got %#v", test.password, test.want, fields)
		}
	}
	request := model.RegisterRequest{
		Username: "tester", Password: strings.Repeat("é", 37), Role: "user",
	}
	if fields := ValidateStruct(request); fields["password"] != "maksimal 72 byte" {
		t.Fatalf("multibyte password byte limit = %#v", fields)
	}
}

func TestCursorRoundTripAndInvalidInput(t *testing.T) {
	createdAt := time.Date(2026, 9, 13, 14, 12, 40, 123, time.FixedZone("test", 7*60*60))
	encoded := EncodeCursor(createdAt, 8)
	cursor, err := DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("DecodeCursor() error = %v", err)
	}
	if !cursor.CreatedAt.Equal(createdAt) || cursor.ID != 8 {
		t.Fatalf("decoded cursor = %#v, want timestamp %v and id 8", cursor, createdAt)
	}
	if _, err := DecodeCursor("not-base64!!"); err == nil {
		t.Fatal("DecodeCursor() accepted malformed input")
	}
}
