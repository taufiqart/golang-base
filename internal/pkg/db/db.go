package db

import (
	"errors"
	"strings"

	"github.com/uptrace/bun/driver/pgdriver"
)

// MapDBError translates PostgreSQL driver errors to user-friendly messages
func MapDBError(err error) error {
	var pgErr pgdriver.Error
	if errors.As(err, &pgErr) {
		code := pgErr.Field('C')
		if pgErr.IntegrityViolation() {
			switch code {
			case "23503":
				return errors.New("referenced resource not found")
			case "23505":
				return errors.New("duplicate value already exists")
			default:
				return errors.New("data integrity error")
			}
		}
		switch code {
		case "22P02":
			return errors.New("invalid input syntax")
		case "22001":
			col := pgErr.Field('c')
			table := pgErr.Field('t')
			msg := pgErr.Field('M')
			if col != "" && table != "" {
				return errors.New("value too long for " + table + "." + col)
			}
			if col != "" {
				return errors.New("value too long for column: " + col)
			}
			if msg != "" {
				return errors.New(msg)
			}
			return errors.New("value too long for column")
		case "22007":
			return errors.New("invalid date or time format")
		case "23502":
			col := pgErr.Field('c')
			table := pgErr.Field('t')
			if col != "" && table != "" {
				return errors.New("required field cannot be empty: " + table + "." + col)
			}
			if col != "" {
				return errors.New("required field cannot be empty: " + col)
			}
			return errors.New("required field cannot be empty")
		case "23514":
			return errors.New("constraint violation")
		}
	}
	return err
}

// IsDBError checks if the error indicates a common DB issue
func IsDBError(err error) bool {
	msg := err.Error()
	return msg == "duplicate value already exists" ||
		msg == "data integrity error" ||
		msg == "referenced resource not found" ||
		msg == "invalid date or time format" ||
		msg == "invalid input syntax" ||
		msg == "constraint violation" ||
		strings.HasPrefix(msg, "value too long") ||
		strings.HasPrefix(msg, "required field cannot be empty")
}

func IsUniqueViolation(err error) bool {
	var pgErr pgdriver.Error
	return errors.As(err, &pgErr) && pgErr.Field('C') == "23505"
}
