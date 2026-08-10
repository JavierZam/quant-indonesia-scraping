package domain

import "errors"

var (
	// ErrNotFound is returned when a requested entity does not exist.
	ErrNotFound = errors.New("entity not found")

	// ErrAlreadyExists is returned when attempting to create a duplicate entity.
	ErrAlreadyExists = errors.New("entity already exists")

	// ErrInvalidInput is returned when input validation fails.
	ErrInvalidInput = errors.New("invalid input")

	// ErrDuplicateURL is returned when a URL has already been ingested.
	ErrDuplicateURL = errors.New("duplicate url: article already ingested")

	// ErrLLMProcessing is returned when the LLM fails to process content.
	ErrLLMProcessing = errors.New("llm processing failed")
)
