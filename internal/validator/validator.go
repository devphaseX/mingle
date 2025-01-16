package validator

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

type Validator struct {
	*validator.Validate
	trans ut.Translator
}

type ValidationErrors struct {
	errors map[string][]string
}

func (v *ValidationErrors) Error() string {
	var builder strings.Builder

	if len(v.errors) == 0 {
		return ""
	}
	_ = json.NewEncoder(&builder).Encode(v.errors)
	return builder.String()
}

func (v *ValidationErrors) FieldErrors() map[string]string {
	firstFieldErrors := make(map[string]string)
	for key, value := range v.errors {
		firstFieldErrors[key] = value[0]
	}

	return firstFieldErrors
}

func New() *Validator {
	validate := validator.New(validator.WithRequiredStructEnabled())
	english := en.New()
	uni := ut.New(english, english)
	trans, _ := uni.GetTranslator("en")

	// Register default translations for English
	_ = en_translations.RegisterDefaultTranslations(validate, trans)

	// Register custom field names
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		jsonTag := fld.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			return fld.Name
		}
		return strings.Split(jsonTag, ",")[0]
	})

	// Register custom error messages
	registerCustomErrorMessages(validate, trans)

	return &Validator{
		Validate: validate,
		trans:    trans,
	}
}

func (v *Validator) Struct(val any) *ValidationErrors {
	if err := v.Validate.Struct(val); err != nil {
		validationErrors := ValidationErrors{
			errors: map[string][]string{},
		}

		for _, entry := range err.(validator.ValidationErrors) {
			fieldName := entry.Field()
			translatedError := entry.Translate(v.trans)
			validationErrors.errors[fieldName] = append(validationErrors.errors[fieldName], translatedError)
		}

		return &validationErrors
	}

	return nil
}

// registerCustomErrorMessages registers custom error messages for validation tags
func registerCustomErrorMessages(validate *validator.Validate, trans ut.Translator) {
	_ = validate.RegisterTranslation("required", trans, func(ut ut.Translator) error {
		return ut.Add("required", "{0} is a required field", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("required", fe.Field())
		return t
	})

	_ = validate.RegisterTranslation("email", trans, func(ut ut.Translator) error {
		return ut.Add("email", "{0} must be a valid email address", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("email", fe.Field())
		return t
	})

}
