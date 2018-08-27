package controller_test

import (
	"testing"

	"code.uber.internal/personal/pourchet/pod-controller"
	"github.com/stretchr/testify/require"
)

func TestLoadPlugin(t *testing.T) {
	t.Run("run_correct_plugin", func(t *testing.T) {
		strat, err := controller.LoadPlugin("./bins/testing.so")
		require.NoError(t, err)
		require.NotNil(t, strat)
	})
	t.Run("plugin_not_found", func(t *testing.T) {
		_, err := controller.LoadPlugin("./bins/does_not_exist")
		require.Error(t, err)
	})
}
