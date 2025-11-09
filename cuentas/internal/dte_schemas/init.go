package dte_schemas

import "fmt"

func Init() error {
	validator, err := NewValidator()
	if err != nil {
		return fmt.Errorf("failed to initialize schema validator: %w", err)
	}
	globalValidator = validator
	return nil
}
