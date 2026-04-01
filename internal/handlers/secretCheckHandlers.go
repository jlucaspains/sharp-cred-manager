package handlers

import (
	"log"
	"net/http"
	"slices"

	"github.com/jlucaspains/sharp-cred-manager/internal/models"
	"github.com/jlucaspains/sharp-cred-manager/internal/services"
)

func (h Handlers) GetSecretList(w http.ResponseWriter, r *http.Request) {
	result := h.SecretList

	h.JSON(w, http.StatusOK, result)
}

func (h Handlers) CheckSecretStatus(w http.ResponseWriter, r *http.Request) {
	name, _ := h.getQueryParam(r, "name")

	log.Println("Received message for name: " + name)

	if name == "" {
		h.JSON(w, http.StatusBadRequest, &models.ErrorResult{Errors: []string{"name is required"}})
		return
	}

	idx := slices.IndexFunc(h.SecretList, func(c models.CheckSecretItem) bool { return c.Name == name })

	if idx < 0 {
		h.JSON(w, http.StatusBadRequest, &models.ErrorResult{Errors: []string{"the provided secret name is not configured"}})
		return
	}

	item := h.SecretList[idx]
	result, err := services.CheckSecretStatus(item, h.SecretWarningValidityDays)

	if err != nil {
		h.JSON(w, http.StatusBadRequest, &models.ErrorResult{Errors: []string{err.Error()}})
		return
	}

	h.JSON(w, http.StatusOK, result)
}
