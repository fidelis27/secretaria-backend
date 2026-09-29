package apperrors

import (
	"errors"
	"net/http"
	"strings"
)

var ErrDatabaseUnavailable = errors.New("database unavailable")

func Classify(err error) error {
	if err == nil {
		return nil
	}

	message := strings.ToLower(err.Error())
	for _, keyword := range []string{
		"ping mariadb",
		"database unavailable",
		"connection refused",
		"no such host",
		"timeout",
		"dial tcp",
		"too many connections",
		"access denied",
		"mysql",
	} {
		if strings.Contains(message, keyword) {
			return ErrDatabaseUnavailable
		}
	}

	return err
}

func IsDatabaseUnavailable(err error) bool {
	return errors.Is(Classify(err), ErrDatabaseUnavailable)
}

func MessageFor(err error) string {
	if err == nil {
		return ""
	}
	if IsDatabaseUnavailable(err) {
		return ErrDatabaseUnavailable.Error()
	}
	return err.Error()
}

func StatusCode(err error) int {
	if IsDatabaseUnavailable(err) {
		return http.StatusFailedDependency
	}
	return http.StatusInternalServerError
}
