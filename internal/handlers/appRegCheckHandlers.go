package handlers

import (
	"log"
	"net/http"
	"slices"

	"github.com/jlucaspains/sharp-cred-manager/internal/models"
	"github.com/jlucaspains/sharp-cred-manager/internal/services"
)

func (h Handlers) GetAppRegList(w http.ResponseWriter, r *http.Request) {
	h.JSON(w, http.StatusOK, h.AppRegList)
}

func (h Handlers) CheckAppRegStatus(w http.ResponseWriter, r *http.Request) {
	name, _ := h.getQueryParam(r, "name")

	log.Println("Received message for app reg name: " + name)

	if name == "" {
		h.JSON(w, http.StatusBadRequest, &models.ErrorResult{Errors: []string{"name is required"}})
		return
	}

	idx := slices.IndexFunc(h.AppRegList, func(a models.CheckAppRegItem) bool { return a.Name == name })

	if idx < 0 {
		h.JSON(w, http.StatusBadRequest, &models.ErrorResult{Errors: []string{"the provided app registration name is not configured"}})
		return
	}

	item := h.AppRegList[idx]
	result, err := services.CheckAppRegStatus(item, h.AppRegWarningValidityDays)

	if err != nil {
		h.JSON(w, http.StatusBadRequest, &models.ErrorResult{Errors: []string{err.Error()}})
		return
	}

	h.JSON(w, http.StatusOK, result)
}
