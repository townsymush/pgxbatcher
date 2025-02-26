package pgxbatcher

import (
	"errors"
)

var (
	ErrEmptyBatch    = errors.New("no queries to execute")
	ErrExecutedBatch = errors.New("this batch has already been executed. Create a new instance or call Reset()")
)

type StatementErrors []error

func (e StatementErrors) Error() string {
	return errors.Join(e...).Error()
}
