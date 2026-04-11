package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/jlucaspains/sharp-cred-manager/internal/models"
)

type Handlers struct {
	CertList                 []models.CheckCertItem
	ExpirationWarningDays    int
	SecretList               []models.CheckSecretItem
	SecretWarningValidityDays int
	CORSOrigins              string
}

func (h Handlers) JSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")

	if len(h.CORSOrigins) > 0 {
		w.Header().Set("Access-Control-Allow-Origin", h.CORSOrigins)
	}

	w.WriteHeader(statusCode)
	// convert data to json
	result, _ := json.Marshal(data)
	w.Write(result)
}

func (h Handlers) HTML(w http.ResponseWriter, statusCode int, data string) {
	w.Header().Set("Content-Type", "text/html")

	if len(h.CORSOrigins) > 0 {
		w.Header().Set("Access-Control-Allow-Origin", h.CORSOrigins)
	}

	w.WriteHeader(statusCode)
	w.Write([]byte(data))
}

func (h Handlers) getQueryParam(r *http.Request, key string) (string, error) {
	if param := r.URL.Query()[key]; param != nil {
		return param[0], nil
	}

	return "", fmt.Errorf("%s not found", key)
}

func (h Handlers) ErrorToHttpResult(err error) (int, *models.ErrorResult) {
	if vErrs, ok := err.(validator.ValidationErrors); ok {
		out := translateErrors(vErrs)
		return http.StatusBadRequest, &models.ErrorResult{Errors: out}
	}

	return http.StatusInternalServerError, &models.ErrorResult{Errors: []string{"Unknown error"}}
}

func translateErrors(err validator.ValidationErrors) []string {
	out := make([]string, len(err))
	for i, fe := range err {
		out[i] = getValidationErrorMsg(fe)
	}
	return out
}

func getValidationErrorMsg(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fe.Field())
	case "lte":
		return fmt.Sprintf("%s should be less than or equal to %s", fe.Field(), fe.Param())
	case "lt":
		return fmt.Sprintf("%s should be less than %s", fe.Field(), fe.Param())
	case "gte":
		return fmt.Sprintf("%s should be greater than or equal to %s", fe.Field(), fe.Param())
	case "gt":
		return fmt.Sprintf("%s should be greater than %s", fe.Field(), fe.Param())
	case "min":
		return fmt.Sprintf("%s should have minimum length of %s", fe.Field(), fe.Param())
	case "max":
		return fmt.Sprintf("%s should have maximum length of %s", fe.Field(), fe.Param())
	case "alpha":
		return fmt.Sprintf("%s should contain alpha characters only", fe.Field())
	}
	return "Unknown error"
}
