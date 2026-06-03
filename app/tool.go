package app

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/validator.v2"
)

// UUID generates a UUID(v7 first) string without hyphens.
func UUID() string {
	var s string
	if id, err := uuid.NewV7(); err != nil {
		s = uuid.New().String() //v4
	} else {
		s = id.String()
	}
	return strings.ReplaceAll(s, "-", "")
}

func ValidateJSON(a any) error {
	// Validate input using validator tags
	return validator.Validate(a)

}

func Now() string {
	return time.Now().Format(time.DateTime)
}
