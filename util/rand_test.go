package util_test

import (
	"testing"

	"github.com/opentarock/service-lobby/util"
	"github.com/stretchr/testify/assert"
)

func TestGeneratedTokenIsOfCorrectLength(t *testing.T) {
	token := util.RandomToken(11)
	assert.Equal(t, 11*2, len(token))
}

func TestTokensGeneratedInSuccessionAreDifferent(t *testing.T) {
	token1 := util.RandomToken(10)
	token2 := util.RandomToken(10)
	assert.NotEqual(t, token1, token2)
}
