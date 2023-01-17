package gora

import (
	"errors"
	"net/http"
	"net/mail"
	"reflect"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

var (
	errUnsupportedType = errors.New("object must be struct, slice, array or pointer to the same data types")
)

type Validator struct {
	validator *validator.Validate
	trans     ut.Translator
}

// returns a new Validator for tagName
// Instantiate a new validator with universal translation based on the 'en' locale.
// See example at https://github.com/go-playground/validator/blob/master/_examples/translations/main.go
func NewValidator(tagName string) *Validator {
	val := validator.New()
	val.SetTagName(tagName)

	enLocale := en.New()
	uni := ut.New(enLocale, enLocale)

	// this is usually know or extracted from http 'Accept-Language' header
	// also see uni.FindTranslator(...)
	trans, _ := uni.GetTranslator("en")
	en_translations.RegisterDefaultTranslations(val, trans)
	return &Validator{
		validator: val,
		trans:     trans,
	}
}

func (val *Validator) SetTagName(tagName string) {
	val.validator.SetTagName(tagName)
}

// Validates structs inside a slice, returns validator.ValidationErrors
// Not that it returns the first encountered error
func (val *Validator) validateSlice(value reflect.Value) error {
	count := value.Len()

	for i := 0; i < count; i++ {
		if err := val.validator.Struct(value.Index(i).Interface()); err != nil {
			return err.(validator.ValidationErrors)
		}
	}

	return nil
}

// Validates structs, pointers to structs and slices/arrays of structs
// Validate will panic if obj is not struct, slice, array or pointers to the same.
func (val *Validator) Validate(obj any) validator.ValidationErrors {
	value := reflect.ValueOf(obj)

	var err error
	switch value.Kind() {
	case reflect.Ptr:
		elem := value.Elem()
		switch reflect.ValueOf(elem.Interface()).Kind() {
		case reflect.Struct:
			err = val.validator.Struct(elem.Interface())
		case reflect.Slice, reflect.Array:
			err = val.validateSlice(elem)
		default:
			panic(errUnsupportedType)
		}
	case reflect.Struct:
		err = val.validator.Struct(value.Interface())
	case reflect.Slice, reflect.Array:
		err = val.validateSlice(value)
	default:
		panic(errUnsupportedType)
	}

	if err != nil {
		return err.(validator.ValidationErrors)
	} else {
		return nil
	}
}

func (val *Validator) TranslateErrors(errs validator.ValidationErrors) validator.ValidationErrorsTranslations {
	return errs.Translate(val.trans)
}

/*
This pattern uses a combination of character sets, character ranges,
and optional groups to match the structure of an email address.
It should match most valid email addresses, including ones with multiple dots
in the domain name and ones with internationalized domain names (IDNs).
*/
func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// Sends translated error messages from go-playground validator using en local
// as JSON.
func (c *Context) ValidationError(err validator.ValidationErrors) {
	errMap := c.validator.TranslateErrors(err)
	c.Status(http.StatusBadRequest).JSON(errMap)
}
