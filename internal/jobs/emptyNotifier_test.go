package jobs

import (
	"testing"

	"github.com/jlucaspains/sharp-cred-manager/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestEmptyNotifier(t *testing.T) {
	emptyNotifier := &EmptyNotifier{}
	err := emptyNotifier.Notify([]models.CertCheckResult{})
	assert.Nil(t, err)
}
