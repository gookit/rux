package util

import "testing"


func TestValidateOptionalSegments(t *testing.T) {
	// Valid paths
	t.Run("valid - optional at end", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Expected no panic, got: %v", r)
			}
		}()
		ValidateOptionalSegments("/posts[/{id}]")
	})

	t.Run("valid - no optional segment", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Expected no panic, got: %v", r)
			}
		}()
		ValidateOptionalSegments("/posts/{id}")
	})

	// Invalid paths - multiple optional
	t.Run("invalid - multiple optional", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for multiple optional segments")
			}
		}()
		ValidateOptionalSegments("/api[/{v1}]/users[/{id}]")
	})

	// Invalid paths - optional not at end
	t.Run("invalid - optional not at end", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for optional segment not at end")
			}
		}()
		ValidateOptionalSegments("/posts[/{category}]/{id}")
	})
}
