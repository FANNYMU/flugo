package dto

import (
	"encoding/json"
	"fmt"
	"net/http"

	"flugo.com/response"
	"flugo.com/validator"
)

func BindJSON(r *http.Request, target interface{}) error {
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}
	return validator.Validate(target)
}

func BindAndValidate(r *http.Request, target interface{}) error {
	return BindJSON(r, target)
}

func HandleValidationError(w http.ResponseWriter, err error) bool {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		response.ValidationError(w, "Validation failed", validationErrors)
		return true
	}
	return false
}

func BindAndRespond(w http.ResponseWriter, r *http.Request, target interface{}) bool {
	if err := BindJSON(r, target); err != nil {
		if !HandleValidationError(w, err) {
			response.BadRequest(w, "Invalid JSON format", err.Error())
		}
		return false
	}
	return true
}

func WriteJSON(w http.ResponseWriter, data interface{}) error {
	response.JSON(w, http.StatusOK, data)
	return nil
}

func WriteError(w http.ResponseWriter, message string, statusCode int) {
	response.Error(w, statusCode, message)
}
