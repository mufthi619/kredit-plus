package response_formatter

import (
	"math"
	"net/http"
)

type Meta struct {
	Page      int   `json:"page"`
	PerPage   int   `json:"per_page"`
	Total     int64 `json:"total"`
	TotalPage int   `json:"total_page"`
}

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
	Errors  []string    `json:"errors,omitempty"`
}

func Success(data interface{}, message string) Response {
	return Response{
		Code:    http.StatusOK,
		Message: message,
		Data:    data,
	}
}

func Created(data interface{}, message string) Response {
	return Response{
		Code:    http.StatusCreated,
		Message: message,
		Data:    data,
	}
}

func Error(code int, message string, errors []string) Response {
	return Response{
		Code:    code,
		Message: message,
		Errors:  errors,
	}
}

func WithPagination(data interface{}, message string, page, perPage int, total int64) Response {
	totalPage := int(math.Ceil(float64(total) / float64(perPage)))

	return Response{
		Code:    http.StatusOK,
		Message: message,
		Data:    data,
		Meta: &Meta{
			Page:      page,
			PerPage:   perPage,
			Total:     total,
			TotalPage: totalPage,
		},
	}
}

func ValidatePagination(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}

func CalculateOffset(page, perPage int) int {
	return (page - 1) * perPage
}
