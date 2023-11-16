package http

import (
	locale "github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/sebasir/rate-limiter-example/model"
	"github.com/sebasir/rate-limiter-example/notification/proto"
)

type CustomValidator struct {
	*validator.Validate
	trans ut.Translator
}

func GetValidator() *CustomValidator {
	val := validator.New()

	isTimeUnitTag := "time-unit"

	if err := val.RegisterValidation(isTimeUnitTag, ValidateTimeUnit); err != nil {
		return nil
	}

	en := locale.New()
	uni := ut.New(en, en)
	trans, _ := uni.GetTranslator("en")

	if err := translations.RegisterDefaultTranslations(val, trans); err != nil {
		return nil
	}

	if err := val.RegisterTranslation(isTimeUnitTag, trans,
		func(ut ut.Translator) (err error) {
			if err = ut.Add(isTimeUnitTag, "{0} must be one of SECOND, MINUTE, HOUR, DAY", false); err != nil {
				return
			}

			return
		}, func(ut ut.Translator, fe validator.FieldError) string {
			t, err := ut.T(fe.Tag(), fe.Field())
			if err != nil {
				return fe.(error).Error()
			}

			return t
		}); err != nil {
		return nil
	}

	val.RegisterStructValidationMapRules(map[string]string{
		"NotificationType": "required",
		"Recipient":        "required,email",
		"Message":          "required",
	}, proto.Notification{})

	return &CustomValidator{
		Validate: val,
		trans:    trans,
	}
}

func (v *CustomValidator) Translate(err error) map[string]string {
	return err.(validator.ValidationErrors).Translate(v.trans)
}

func (v *CustomValidator) RegisterStructValidationMapRules(rules map[string]string, types ...interface{}) {
	v.Validate.RegisterStructValidationMapRules(rules, types...)
}

func ValidateTimeUnit(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	_, exists := model.TimeUnitMap[val]
	return exists
}
