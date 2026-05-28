// Package utils provides the standardised JSON response envelope used by all
// handlers.  Every response — success or error — passes through these helpers
// to guarantee a consistent shape for the frontend to parse.
package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ResponseMeta carries pagination / supplemental metadata in success responses.
type ResponseMeta struct {
	Page     int    `json:"page,omitempty"`
	PageSize int    `json:"page_size,omitempty"`
	Total    int64  `json:"total,omitempty"`
	Message  string `json:"message,omitempty"`
}

// APIError carries structured error information.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

// Success writes a 200 OK JSON response with the standard envelope.
func Success(c *gin.Context, data interface{}, meta *ResponseMeta) {
	body := gin.H{"success": true, "data": data}
	if meta != nil {
		body["meta"] = meta
	}
	c.JSON(http.StatusOK, body)
}

// Created writes a 201 Created JSON response.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": data})
}

// NoContent writes a 204 No Content response.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// ErrorResponse writes a structured error JSON response.
func ErrorResponse(c *gin.Context, httpStatus int, code, message, field string) {
	c.JSON(httpStatus, gin.H{
		"success": false,
		"error": APIError{
			Code:    code,
			Message: message,
			Field:   field,
		},
	})
}

// ValidationError is a convenience wrapper for 400 validation failures.
func ValidationError(c *gin.Context, err error) {
	ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), "")
}

// BadRequest writes a 400 response.
func BadRequest(c *gin.Context, message, code string) {
	ErrorResponse(c, http.StatusBadRequest, code, message, "")
}

// Unauthorized writes a 401 response.
func Unauthorized(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", message, "")
}

// Forbidden writes a 403 response.
func Forbidden(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusForbidden, "FORBIDDEN", message, "")
}

// NotFound writes a 404 response.
func NotFound(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", message, "")
}

// RateLimited writes a 429 response.
func RateLimited(c *gin.Context) {
	ErrorResponse(c, http.StatusTooManyRequests, "RATE_LIMITED", "Too many requests. Please try again later.", "")
}

// IntegrationError writes a 502 response for third-party failures.
func IntegrationError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusBadGateway, "INTEGRATION_ERROR", message, "")
}

// InternalError writes a 500 response.
func InternalError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", message, "")
}
