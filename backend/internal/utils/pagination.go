// Package utils provides pagination helpers for list endpoints.
package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

// PaginationParams holds parsed page / page_size query parameters.
type PaginationParams struct {
	Page     int
	PageSize int
	Offset   int
}

// ParsePagination reads page and page_size from the query string with safe defaults.
func ParsePagination(c *gin.Context) PaginationParams {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = defaultPage
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
		Offset:   (page - 1) * pageSize,
	}
}

// NewMeta builds a ResponseMeta from pagination params and total count.
func NewMeta(p PaginationParams, total int64) *ResponseMeta {
	return &ResponseMeta{
		Page:     p.Page,
		PageSize: p.PageSize,
		Total:    total,
	}
}
