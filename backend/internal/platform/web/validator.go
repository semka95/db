package web

import (
	"reflect"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
)

// AppValidator represents validation struct
type AppValidator struct {
	UniTrans   *ut.UniversalTranslator
	V          *validator.Validate
	Translator ut.Translator
}

// NewAppValidator will initialize validator with translator
func NewAppValidator() (*AppValidator, error) {
	av := new(AppValidator)
	translator := en.New()
	av.UniTrans = ut.New(translator, translator)
	var found bool
	av.Translator, found = av.UniTrans.GetTranslator("en")
	if !found {
		av.Translator = av.UniTrans.GetFallback()
	}

	av.V = validator.New()

	err := enTranslations.RegisterDefaultTranslations(av.V, av.Translator)
	if err != nil {
		return nil, err
	}

	av.V.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return av, nil
}

// Validate serving to be called by Echo to validate url
func (av *AppValidator) Validate(i interface{}) error {
	return av.V.Struct(i)
}
