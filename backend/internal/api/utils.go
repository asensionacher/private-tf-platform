package api

import "github.com/google/uuid"

// generateID generates a new UUID for use as an identifier
func generateID() string {
	return uuid.New().String()
}
