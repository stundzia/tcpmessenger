package messenger

import "fmt"

type nameTakenError struct {
	errString string
}

func NewNameTakenError(name string) error {
	return &nameTakenError{errString: fmt.Sprintf("%s is already taken", name)}
}

func (err *nameTakenError) Error() string {
	return err.errString
}