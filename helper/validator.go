package helper

import (
	"errors"
	"reflect"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New()
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return field.Name
		}
		return name
	})
	_ = v.RegisterValidation("username", func(fl validator.FieldLevel) bool {
		for _, r := range fl.Field().String() {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '.' && r != '_' {
				return false
			}
		}
		return true
	})
	_ = v.RegisterValidation("nim", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		if len(value) < 3 || len(value) > 20 {
			return false
		}
		for _, r := range value {
			if r < '0' || r > '9' {
				return false
			}
		}
		return true
	})
	_ = v.RegisterValidation("nospace", func(fl validator.FieldLevel) bool {
		return !strings.ContainsAny(fl.Field().String(), " \t\n\r")
	})
	_ = v.RegisterValidation("strongpassword", func(fl validator.FieldLevel) bool {
		return passwordStrength(fl.Field().String()) == ""
	})
	_ = v.RegisterValidation("passwordbytes", func(fl validator.FieldLevel) bool {
		return len(fl.Field().String()) <= 72
	})
	return v
}

func ValidateStruct(value any) map[string]string {
	err := validate.Struct(value)
	if err == nil {
		return nil
	}
	var invalid *validator.InvalidValidationError
	if errors.As(err, &invalid) {
		return map[string]string{"_": "objek yang divalidasi tidak sah"}
	}
	var fieldErrors validator.ValidationErrors
	if !errors.As(err, &fieldErrors) {
		return map[string]string{"_": "validasi gagal"}
	}
	fields := make(map[string]string, len(fieldErrors))
	for _, fieldErr := range fieldErrors {
		if _, exists := fields[fieldErr.Field()]; !exists {
			fields[fieldErr.Field()] = validationMessage(fieldErr)
		}
	}
	return fields
}

func validationMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "wajib diisi"
	case "email":
		return "format email tidak valid"
	case "min":
		if err.Kind() == reflect.String {
			return "minimal " + err.Param() + " karakter"
		}
		return "nilai minimal " + err.Param()
	case "max":
		if err.Kind() == reflect.String {
			return "maksimal " + err.Param() + " karakter"
		}
		return "nilai maksimal " + err.Param()
	case "gte":
		return "nilai minimal " + err.Param()
	case "lte":
		return "nilai maksimal " + err.Param()
	case "alphanum":
		return "hanya boleh berisi huruf dan angka"
	case "username":
		return "hanya boleh huruf, angka, titik, dan garis bawah"
	case "nim":
		return "NIM harus terdiri dari 3 sampai 20 digit"
	case "nospace":
		return "tidak boleh mengandung spasi"
	case "strongpassword":
		if value, ok := err.Value().(string); ok {
			return passwordStrength(value)
		}
		return "password tidak memenuhi syarat"
	case "passwordbytes":
		return "maksimal 72 byte"
	case "oneof":
		return "harus salah satu dari: " + strings.ReplaceAll(err.Param(), " ", ", ")
	default:
		return "tidak memenuhi aturan " + err.Tag()
	}
}

func passwordStrength(password string) string {
	switch strings.ToLower(password) {
	case "password123", "password1", "12345678", "qwerty123":
		return "password terlalu umum"
	}
	hasLetter, hasDigit := false, false
	for _, r := range password {
		hasLetter = hasLetter || unicode.IsLetter(r)
		hasDigit = hasDigit || unicode.IsDigit(r)
	}
	if !hasLetter || !hasDigit {
		return "harus memuat huruf dan angka"
	}
	return ""
}
