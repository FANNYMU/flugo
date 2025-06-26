package response

import (
	"encoding/json"
	"net/http"
	"time"
)

type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Errors    interface{} `json:"errors,omitempty"`
	Meta      *Meta       `json:"meta,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

type Meta struct {
	Page       int `json:"page,omitempty"`
	PerPage    int `json:"per_page,omitempty"`
	Total      int `json:"total,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

type PaginatedResponse struct {
	Data []interface{} `json:"data"`
	Meta Meta          `json:"meta"`
}

func writeJSON(w http.ResponseWriter, statusCode int, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response.Timestamp = time.Now()

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(response)
}

func Success(w http.ResponseWriter, data interface{}, message ...string) {
	msg := "Success"
	if len(message) > 0 {
		msg = message[0]
	}

	response := APIResponse{
		Success: true,
		Message: msg,
		Data:    data,
	}

	writeJSON(w, http.StatusOK, response)
}

func Created(w http.ResponseWriter, data interface{}, message ...string) {
	msg := "Resource created successfully"
	if len(message) > 0 {
		msg = message[0]
	}

	response := APIResponse{
		Success: true,
		Message: msg,
		Data:    data,
	}

	writeJSON(w, http.StatusCreated, response)
}

func Updated(w http.ResponseWriter, data interface{}, message ...string) {
	msg := "Resource updated successfully"
	if len(message) > 0 {
		msg = message[0]
	}

	response := APIResponse{
		Success: true,
		Message: msg,
		Data:    data,
	}

	writeJSON(w, http.StatusOK, response)
}

func Deleted(w http.ResponseWriter, message ...string) {
	msg := "Resource deleted successfully"
	if len(message) > 0 {
		msg = message[0]
	}

	response := APIResponse{
		Success: true,
		Message: msg,
	}

	writeJSON(w, http.StatusOK, response)
}

func Paginated(w http.ResponseWriter, data []interface{}, meta Meta, message ...string) {
	msg := "Data retrieved successfully"
	if len(message) > 0 {
		msg = message[0]
	}

	paginatedData := PaginatedResponse{
		Data: data,
		Meta: meta,
	}

	response := APIResponse{
		Success: true,
		Message: msg,
		Data:    paginatedData,
	}

	writeJSON(w, http.StatusOK, response)
}

func Error(w http.ResponseWriter, statusCode int, message string, errors ...interface{}) {
	response := APIResponse{
		Success: false,
		Message: message,
	}

	if len(errors) > 0 {
		response.Errors = errors[0]
	}

	writeJSON(w, statusCode, response)
}

func BadRequest(w http.ResponseWriter, message string, errors ...interface{}) {
	Error(w, http.StatusBadRequest, message, errors...)
}

func Unauthorized(w http.ResponseWriter, message ...string) {
	msg := "Authentication required"
	if len(message) > 0 {
		msg = message[0]
	}
	Error(w, http.StatusUnauthorized, msg)
}

func Forbidden(w http.ResponseWriter, message ...string) {
	msg := "Access forbidden"
	if len(message) > 0 {
		msg = message[0]
	}
	Error(w, http.StatusForbidden, msg)
}

func NotFound(w http.ResponseWriter, message ...string) {
	msg := "Resource not found"
	if len(message) > 0 {
		msg = message[0]
	}
	Error(w, http.StatusNotFound, msg)
}

func Conflict(w http.ResponseWriter, message string, errors ...interface{}) {
	Error(w, http.StatusConflict, message, errors...)
}

func ValidationError(w http.ResponseWriter, message string, errors interface{}) {
	response := APIResponse{
		Success: false,
		Message: message,
		Errors:  errors,
	}

	writeJSON(w, http.StatusUnprocessableEntity, response)
}

func InternalError(w http.ResponseWriter, message ...string) {
	msg := "Internal server error"
	if len(message) > 0 {
		msg = message[0]
	}
	Error(w, http.StatusInternalServerError, msg)
}

func ServiceUnavailable(w http.ResponseWriter, message ...string) {
	msg := "Service temporarily unavailable"
	if len(message) > 0 {
		msg = message[0]
	}
	Error(w, http.StatusServiceUnavailable, msg)
}

func TooManyRequests(w http.ResponseWriter, message ...string) {
	msg := "Too many requests"
	if len(message) > 0 {
		msg = message[0]
	}
	Error(w, http.StatusTooManyRequests, msg)
}

func Custom(w http.ResponseWriter, statusCode int, success bool, message string, data interface{}, errors interface{}) {
	response := APIResponse{
		Success: success,
		Message: message,
		Data:    data,
		Errors:  errors,
	}

	writeJSON(w, statusCode, response)
}

func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(data)
}

func EmptySuccess(w http.ResponseWriter) {
	Success(w, nil, "Operation completed successfully")
}

func SuccessWithMeta(w http.ResponseWriter, data interface{}, meta *Meta, message ...string) {
	msg := "Success"
	if len(message) > 0 {
		msg = message[0]
	}

	response := APIResponse{
		Success: true,
		Message: msg,
		Data:    data,
		Meta:    meta,
	}

	writeJSON(w, http.StatusOK, response)
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version,omitempty"`
	Services  map[string]string `json:"services,omitempty"`
}

func Health(w http.ResponseWriter, status string, version string, services map[string]string) {
	health := HealthStatus{
		Status:    status,
		Timestamp: time.Now(),
		Version:   version,
		Services:  services,
	}

	statusCode := http.StatusOK
	if status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	JSON(w, statusCode, health)
}

func BindJSON(r *http.Request, target interface{}) error {
	return json.NewDecoder(r.Body).Decode(target)
}
