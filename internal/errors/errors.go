package errors

import "fmt"

// CustomError représente une erreur personnalisée pour l'application.
type CustomError struct {
    Code    int
    Message string
}

// New crée une nouvelle instance de CustomError.
func New(code int, message string) *CustomError {
    return &CustomError{
        Code:    code,
        Message: message,
    }
}

// Error implémente l'interface error pour CustomError.
func (e *CustomError) Error() string {
    return fmt.Sprintf("Code %d: %s", e.Code, e.Message)
}