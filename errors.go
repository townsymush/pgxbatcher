package pgxbatcher

import "fmt"

type StatementError struct {
	sql string
	err error
}

type StatementErrors []StatementError

func (b StatementErrors) Error() string {
	errString := ""
	if len(b) > 0 {
		for _, v := range b {
			errString += v.Error() + "\n "
		}
		return errString
	}
	return errString
}

func (b StatementErrors) isErrors() bool {
	return len(b) > 0
}

func (b StatementError) Error() string {
	return fmt.Sprintf("sql: %s, %s", b.sql, b.err.Error())
}
