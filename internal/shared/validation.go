package shared

import (
	"encoding/json"
	"net/http"
)

func DecodeAndValidate(r *http.Request, dst interface{}) *AppError {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return NewBadRequestError("corpo da requisição inválido", err)
	}
	return nil
}

func IsValidMatricula(matricula string) bool {
	if len(matricula) < 6 {
		return false
	}
	for _, c := range matricula {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func IsValidSemester(semester int) bool {
	return semester >= 1 && semester <= 8
}
