package model

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type FieldErrors []FieldError

func (f *FieldErrors) Add(field string, err error) {
	if err != nil {
		*f = append(*f, FieldError{Field: field, Message: err.Error()})
	}
}

type ValidationError struct {
	Details FieldErrors
}

func (e *ValidationError) Error() string {
	return "invalid input"
}

func (e *ValidationError) Unwrap() error {
	return ErrInvalidInput
}
