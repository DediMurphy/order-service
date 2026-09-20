package domain

import "net/http"

type AppError struct {
	Status int    
	Code   string 
	Detail string 
}

func (e *AppError) Error() string {
	if e.Detail == "" {
		return e.Code
	}
	return e.Code + ": " + e.Detail
}

func NewBadRequest(code, detail string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Code: code, Detail: detail}
}

func NewNotFound(detail string) *AppError {
	return &AppError{Status: http.StatusNotFound, Code: "not found", Detail: detail}
}

func NewConflict(code, detail string) *AppError {
	return &AppError{Status: http.StatusConflict, Code: code, Detail: detail}
}
