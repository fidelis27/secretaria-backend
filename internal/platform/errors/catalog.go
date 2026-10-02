package apperrors

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-sql-driver/mysql"
)

var ErrDatabaseUnavailable = errors.New("database unavailable")
var ErrDuplicateEmail = errors.New("e-mail ja cadastrado")
var ErrDuplicateMembership = errors.New("usuario ja pertence ao grupo")
var ErrDuplicateGroupName = errors.New("ja existe um grupo com esse nome nessa instituicao")
var ErrForeignKeyViolation = errors.New("referencia informada nao existe")

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

func IsDuplicateEntry(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

func DuplicateEntryMessage(err error) string {
	if !IsDuplicateEntry(err) {
		return ""
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "create membership"), strings.Contains(message, "member_groups"):
		return ErrDuplicateMembership.Error()
	case strings.Contains(message, "create group"), strings.Contains(message, "groups"):
		return ErrDuplicateGroupName.Error()
	case strings.Contains(message, "create user"), strings.Contains(message, "email"):
		return ErrDuplicateEmail.Error()
	default:
		return "registro duplicado"
	}
}

func IsForeignKeyViolation(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && (mysqlErr.Number == 1451 || mysqlErr.Number == 1452)
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
	if IsDuplicateEntry(err) {
		return http.StatusConflict
	}
	if IsForeignKeyViolation(err) {
		return http.StatusUnprocessableEntity
	}
	return http.StatusInternalServerError
}
