package utils

import (
	
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashAndCheck(t *testing.T) {
	// GIVEN
	pw := "S3cr3t!!"

	// WHEN
	hash, err := HashPassword(pw)
	require.NoError(t, err)
	require.NotEmpty(t, hash)

	// THEN
	assert.True(t, CheckPassword(hash, pw))
	assert.False(t, CheckPassword(hash, "wrong"))
}

