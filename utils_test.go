package controller

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterErrors(t *testing.T) {
	t.Run("average_case", func(t *testing.T) {
		errs := []error{nil, io.EOF, nil}
		filtered := filterErrors(errs)
		require.Lenf(t, filtered, 1, "Wrong slice length")
		require.Equal(t, io.EOF, filtered[0])
	})
	t.Run("no_errors", func(t *testing.T) {
		errs := []error{nil, nil}
		filtered := filterErrors(errs)
		require.Lenf(t, filtered, 0, "Wrong slice length")
	})
	t.Run("all_errors", func(t *testing.T) {
		errs := []error{io.EOF, io.EOF, io.EOF}
		filtered := filterErrors(errs)
		require.Lenf(t, filtered, 3, "Wrong slice length")
		require.Equal(t, io.EOF, filtered[0])
		require.Equal(t, io.EOF, filtered[1])
		require.Equal(t, io.EOF, filtered[2])
	})
}
