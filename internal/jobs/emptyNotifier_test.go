package jobs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmptyNotifier(t *testing.T) {
	emptyNotifier := &EmptyNotifier{}
	err := emptyNotifier.Notify([]CheckNotification{})
	assert.Nil(t, err)
}
