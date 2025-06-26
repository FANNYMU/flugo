package validator

import (
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Tag     string `json:"tag"`
	Value   string `json:"value"`
}

type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	var messages []string
	for _, err := range v {
		messages = append(messages, fmt.Sprintf("%s: %s", err.Field, err.Message))
	}
	return strings.Join(messages, ", ")
}

func (v ValidationErrors) HasErrors() bool {
	return len(v) > 0
}

type Validator struct {
	customValidators map[string]func(interface{}) bool
	customMessages   map[string]string
}

func New() *Validator {
	return &Validator{
		customValidators: make(map[string]func(interface{}) bool),
		customMessages:   make(map[string]string),
	}
}

var DefaultValidator = New()

func (v *Validator) RegisterCustom(tag string, validator func(interface{}) bool, message string) {
	v.customValidators[tag] = validator
	v.customMessages[tag] = message
}

func Validate(target interface{}) error {
	return DefaultValidator.Validate(target)
}

func (v *Validator) Validate(target interface{}) error {
	var errors ValidationErrors

	val := reflect.ValueOf(target)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a struct or pointer to struct")
	}

	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		if !fieldValue.CanInterface() {
			continue
		}

		fieldErrors := v.validateField(field, fieldValue)
		errors = append(errors, fieldErrors...)
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

func (v *Validator) validateField(field reflect.StructField, value reflect.Value) []ValidationError {
	var errors []ValidationError
	tag := field.Tag
	fieldName := field.Name

	if jsonTag := tag.Get("json"); jsonTag != "" && jsonTag != "-" {
		parts := strings.Split(jsonTag, ",")
		if parts[0] != "" {
			fieldName = parts[0]
		}
	}

	fieldInterface := value.Interface()
	fieldStr := fmt.Sprintf("%v", fieldInterface)

	// Required validation
	if tag.Get("required") == "true" {
		if v.isZeroValue(value) {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Message: "field is required",
				Tag:     "required",
				Value:   fieldStr,
			})
			return errors
		}
	}

	if v.isZeroValue(value) {
		return errors
	}

	if value.Kind() == reflect.String {
		strValue := value.String()

		if minLenStr := tag.Get("min_length"); minLenStr != "" {
			if minLen, err := strconv.Atoi(minLenStr); err == nil {
				if len(strValue) < minLen {
					errors = append(errors, ValidationError{
						Field:   fieldName,
						Message: fmt.Sprintf("minimum length is %d characters", minLen),
						Tag:     "min_length",
						Value:   fieldStr,
					})
				}
			}
		}

		if maxLenStr := tag.Get("max_length"); maxLenStr != "" {
			if maxLen, err := strconv.Atoi(maxLenStr); err == nil {
				if len(strValue) > maxLen {
					errors = append(errors, ValidationError{
						Field:   fieldName,
						Message: fmt.Sprintf("maximum length is %d characters", maxLen),
						Tag:     "max_length",
						Value:   fieldStr,
					})
				}
			}
		}

		if tag.Get("email") == "true" {
			if !v.isValidEmail(strValue) {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: "must be a valid email address",
					Tag:     "email",
					Value:   fieldStr,
				})
			}
		}

		if tag.Get("url") == "true" {
			if !v.isValidURL(strValue) {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: "must be a valid URL",
					Tag:     "url",
					Value:   fieldStr,
				})
			}
		}

		if tag.Get("phone") == "true" {
			if !v.isValidPhone(strValue) {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: "must be a valid phone number",
					Tag:     "phone",
					Value:   fieldStr,
				})
			}
		}

		if tag.Get("alphanumeric") == "true" {
			if !v.isAlphanumeric(strValue) {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: "must contain only letters and numbers",
					Tag:     "alphanumeric",
					Value:   fieldStr,
				})
			}
		}

		if tag.Get("alpha") == "true" {
			if !v.isAlpha(strValue) {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: "must contain only letters",
					Tag:     "alpha",
					Value:   fieldStr,
				})
			}
		}

		if tag.Get("numeric") == "true" {
			if !v.isNumeric(strValue) {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: "must contain only numbers",
					Tag:     "numeric",
					Value:   fieldStr,
				})
			}
		}

		if tag.Get("ip") == "true" {
			if !v.isValidIP(strValue) {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: "must be a valid IP address",
					Tag:     "ip",
					Value:   fieldStr,
				})
			}
		}

		if dateFormat := tag.Get("date"); dateFormat != "" {
			if !v.isValidDate(strValue, dateFormat) {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("must be a valid date in format %s", dateFormat),
					Tag:     "date",
					Value:   fieldStr,
				})
			}
		}

		if regexPattern := tag.Get("regex"); regexPattern != "" {
			if !v.matchesRegex(strValue, regexPattern) {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: "does not match required pattern",
					Tag:     "regex",
					Value:   fieldStr,
				})
			}
		}

		if enumValues := tag.Get("enum"); enumValues != "" {
			if !v.isInEnum(strValue, enumValues) {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("must be one of: %s", enumValues),
					Tag:     "enum",
					Value:   fieldStr,
				})
			}
		}
	}

	if v.isNumericType(value) {
		numValue := v.getNumericValue(value)

		if minStr := tag.Get("min"); minStr != "" {
			if min, err := strconv.ParseFloat(minStr, 64); err == nil {
				if numValue < min {
					errors = append(errors, ValidationError{
						Field:   fieldName,
						Message: fmt.Sprintf("minimum value is %v", min),
						Tag:     "min",
						Value:   fieldStr,
					})
				}
			}
		}

		if maxStr := tag.Get("max"); maxStr != "" {
			if max, err := strconv.ParseFloat(maxStr, 64); err == nil {
				if numValue > max {
					errors = append(errors, ValidationError{
						Field:   fieldName,
						Message: fmt.Sprintf("maximum value is %v", max),
						Tag:     "max",
						Value:   fieldStr,
					})
				}
			}
		}
	}

	if value.Kind() == reflect.Slice || value.Kind() == reflect.Array {
		if minItemsStr := tag.Get("min_items"); minItemsStr != "" {
			if minItems, err := strconv.Atoi(minItemsStr); err == nil {
				if value.Len() < minItems {
					errors = append(errors, ValidationError{
						Field:   fieldName,
						Message: fmt.Sprintf("minimum items is %d", minItems),
						Tag:     "min_items",
						Value:   fieldStr,
					})
				}
			}
		}

		if maxItemsStr := tag.Get("max_items"); maxItemsStr != "" {
			if maxItems, err := strconv.Atoi(maxItemsStr); err == nil {
				if value.Len() > maxItems {
					errors = append(errors, ValidationError{
						Field:   fieldName,
						Message: fmt.Sprintf("maximum items is %d", maxItems),
						Tag:     "max_items",
						Value:   fieldStr,
					})
				}
			}
		}
	}

	for tag, validator := range v.customValidators {
		if field.Tag.Get(tag) == "true" {
			if !validator(fieldInterface) {
				message := v.customMessages[tag]
				if message == "" {
					message = fmt.Sprintf("failed custom validation: %s", tag)
				}
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: message,
					Tag:     tag,
					Value:   fieldStr,
				})
			}
		}
	}

	return errors
}

func (v *Validator) isZeroValue(val reflect.Value) bool {
	switch val.Kind() {
	case reflect.String:
		return val.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return val.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return val.Float() == 0
	case reflect.Bool:
		return !val.Bool()
	case reflect.Ptr, reflect.Interface:
		return val.IsNil()
	case reflect.Slice, reflect.Map, reflect.Array:
		return val.Len() == 0
	default:
		return false
	}
}

func (v *Validator) isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func (v *Validator) isValidURL(urlStr string) bool {
	_, err := url.ParseRequestURI(urlStr)
	return err == nil
}

func (v *Validator) isValidPhone(phone string) bool {
	phoneRegex := regexp.MustCompile(`^[\+]?[\d\s\-\(\)]{7,15}$`)
	return phoneRegex.MatchString(phone)
}

func (v *Validator) isAlphanumeric(str string) bool {
	alphanumericRegex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	return alphanumericRegex.MatchString(str)
}

func (v *Validator) isAlpha(str string) bool {
	alphaRegex := regexp.MustCompile(`^[a-zA-Z]+$`)
	return alphaRegex.MatchString(str)
}

func (v *Validator) isNumeric(str string) bool {
	numericRegex := regexp.MustCompile(`^[0-9]+$`)
	return numericRegex.MatchString(str)
}

func (v *Validator) isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

func (v *Validator) isValidDate(dateStr, format string) bool {
	_, err := time.Parse(format, dateStr)
	return err == nil
}

func (v *Validator) matchesRegex(str, pattern string) bool {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return regex.MatchString(str)
}

func (v *Validator) isInEnum(value, enumStr string) bool {
	enumValues := strings.Split(enumStr, ",")
	for _, enumValue := range enumValues {
		if strings.TrimSpace(enumValue) == value {
			return true
		}
	}
	return false
}

func (v *Validator) isNumericType(val reflect.Value) bool {
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func (v *Validator) getNumericValue(val reflect.Value) float64 {
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(val.Uint())
	case reflect.Float32, reflect.Float64:
		return val.Float()
	default:
		return 0
	}
}

func RegisterCustom(tag string, validator func(interface{}) bool, message string) {
	DefaultValidator.RegisterCustom(tag, validator, message)
}

func InitValidators() {
	// Initialize validators (alias for backward compatibility)
}
