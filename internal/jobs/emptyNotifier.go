package jobs

import "github.com/jlucaspains/sharp-cred-manager/internal/models"

type EmptyNotifier struct{}

func (m *EmptyNotifier) Notify(result []models.CertCheckResult) error {
	return nil
}

func (m *EmptyNotifier) IsReady() bool {
	return true
}
