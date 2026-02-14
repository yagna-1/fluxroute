package adapters

import "errors"

var (
	ErrMissingAPIKey = errors.New("missing api key")
	ErrEmptyPrompt   = errors.New("prompt is empty")
)
