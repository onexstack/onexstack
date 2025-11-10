// Package binding provides enhanced binding utilities that extend Gin's binding capabilities.
// It solves the common problem of binding data from multiple sources (URI, JSON, etc.)
// without validation errors caused by partially filled structs.
package binding

import (
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// Internal package variables
var (
	// originalValidator stores the original Gin validator for use in final validation
	originalValidator binding.StructValidator

	// initOnce ensures the validator is captured only once
	initOnce sync.Once
)

// init captures the original validator and sets the global validator to nil.
// This happens at package import time and only once.
func init() {
	// Use Once to ensure this only happens once even if multiple goroutines enter simultaneously
	initOnce.Do(func() {
		// Save original validator
		originalValidator = binding.Validator

		// Set global validator to nil to skip validation during binding
		binding.Validator = nil
	})
}

// Bind processes data from multiple sources without validating between each step,
// then performs a single validation at the end.
//
// This solves the problem where binding from one source (e.g., URI) fails validation
// because fields from another source (e.g., JSON) haven't been bound yet.
//
// Example usage:
//
//	var req UserUpdateRequest
//	if err := binding.Bind(c, &req, binding.URI, binding.JSON); err != nil {
//	    c.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
//	    return
//	}
func Bind(c *gin.Context, obj interface{}, bindFuncs ...func(*gin.Context, interface{}) error) error {
	// Execute all binding functions (validation is already disabled)
	for _, bindFunc := range bindFuncs {
		if err := bindFunc(c, obj); err != nil {
			return err // Returns parsing errors but not validation errors
		}
	}

	// Perform validation manually after all bindings are complete
	if originalValidator != nil {
		return originalValidator.ValidateStruct(obj)
	}

	return nil
}

// Common binding functions for use with Bind

// URI binds URI parameters to the given object.
// Uses Gin's ShouldBindUri but without validation.
func URI(c *gin.Context, obj interface{}) error {
	return c.ShouldBindUri(obj)
}

// JSON binds JSON body to the given object.
// Uses Gin's ShouldBindJSON but without validation.
func JSON(c *gin.Context, obj interface{}) error {
	return c.ShouldBindJSON(obj)
}
