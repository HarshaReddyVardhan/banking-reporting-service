package domain

import "errors"

var (
	ErrReportNotFound = errors.New("report not found")
	ErrInvalidRequest = errors.New("invalid report request")
	ErrInternalServer = errors.New("internal server error")
)
