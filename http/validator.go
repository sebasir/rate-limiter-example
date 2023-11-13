package http

import (
	locale "github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/sebasir/rate-limiter-example/notification/proto"
)

type Validator struct {
	*validator.Validate
	trans ut.Translator
}

func GetValidator() *Validator {
	val := validator.New()

	en := locale.New()
	uni := ut.New(en, en)
	trans, _ := uni.GetTranslator("en")

	err := translations.RegisterDefaultTranslations(val, trans)
	if err != nil {
		return nil
	}

	val.RegisterStructValidationMapRules(map[string]string{
		"NotificationType": "required",
		"Recipient":        "required,email",
		"Message":          "required",
	}, proto.Notification{})

	return &Validator{
		Validate: val,
		trans:    trans,
	}
}

func (v *Validator) Translate(err error) map[string]string {
	return err.(validator.ValidationErrors).Translate(v.trans)
}

func (v *Validator) RegisterStructValidationMapRules(rules map[string]string, types ...interface{}) {
	v.Validate.RegisterStructValidationMapRules(rules, types...)
}
