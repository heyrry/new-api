package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestValidateChannelRpmLimit(t *testing.T) {
	baseURL := "https://new-api.example.com"
	valid := 1
	channel := &model.Channel{Type: 3, Key: "secret", BaseURL: &baseURL, RpmLimit: &valid}
	require.NoError(t, validateChannel(channel, false))

	zero := 0
	channel.RpmLimit = &zero
	require.ErrorContains(t, validateChannel(channel, false), "rpm_limit must be a positive integer")

	negative := -1
	channel.RpmLimit = &negative
	require.ErrorContains(t, validateChannel(channel, false), "rpm_limit must be a positive integer")

	channel.RpmLimit = nil
	require.NoError(t, validateChannel(channel, false))
}
