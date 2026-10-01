package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetEnvAsDuration(t *testing.T) {
	t.Setenv("TEST_DURATION", "45m")
	assert.Equal(t, 45*time.Minute, GetEnvAsDuration("TEST_DURATION", time.Minute))

	for _, invalid := range []string{"", "abc", "-5m", "0s"} {
		t.Setenv("TEST_DURATION", invalid)
		assert.Equal(t, time.Minute, GetEnvAsDuration("TEST_DURATION", time.Minute), invalid)
	}
}
