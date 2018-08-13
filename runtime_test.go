package controller_test

import (
	"local/controller"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadPlugin(t *testing.T) {
	strat, err := controller.LoadPlugin("./bins/testing.so")
	require.NoError(t, err)
	require.NotNil(t, strat)
}
