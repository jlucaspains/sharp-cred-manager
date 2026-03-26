package handlers

import (
	"log"
	"net/http"
	"slices"

	"github.com/jlucaspains/sharp-cred-manager/internal/models"
	"github.com/jlucaspains/sharp-cred-manager/internal/services"
)

func (h Handlers) GetCertList(w http.ResponseWriter, r *http.Request) {
	result := h.CertList

	h.JSON(w, http.StatusOK, result)
}

func (h Handlers) CheckCertStatus(w http.ResponseWriter, r *http.Request) {
	name, _ := h.getQueryParam(r, "name")

	log.Println("Received message for name: " + name)

	if name == "" {
		h.JSON(w, http.StatusBadRequest, &models.ErrorResult{Errors: []string{"name is required"}})
		return
	}

	idx := slices.IndexFunc(h.CertList, func(c models.CheckCertItem) bool { return c.Name == name })

	if idx < 0 {
		h.JSON(w, http.StatusBadRequest, &models.ErrorResult{Errors: []string{"the provided cert name is not configured"}})
		return
	}

	item := h.CertList[idx]
	result, err := services.CheckCertStatus(item, h.ExpirationWarningDays)

	if err != nil {
		h.JSON(w, http.StatusBadRequest, &models.ErrorResult{Errors: []string{err.Error()}})
		return
	}

	h.JSON(w, http.StatusOK, result)
}
