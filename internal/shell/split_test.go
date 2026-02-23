package shell_test

import (
	"testing"

	"github.com/k0sproject/k0sctl/internal/shell"
	"github.com/stretchr/testify/require"
)

func TestSplit(t *testing.T) {
	t.Run("simple words", func(t *testing.T) {
		result, err := shell.Split("hello world")
		require.NoError(t, err)
		require.Equal(t, []string{"hello", "world"}, result)
	})

	t.Run("empty string", func(t *testing.T) {
		result, err := shell.Split("")
		require.NoError(t, err)
		require.Empty(t, result)
	})

	t.Run("double quoted string", func(t *testing.T) {
		result, err := shell.Split(`"hello world" foo`)
		require.NoError(t, err)
		require.Equal(t, []string{"hello world", "foo"}, result)
	})

	t.Run("single quoted string", func(t *testing.T) {
		result, err := shell.Split(`'hello world' foo`)
		require.NoError(t, err)
		require.Equal(t, []string{"hello world", "foo"}, result)
	})

	t.Run("escaped space", func(t *testing.T) {
		result, err := shell.Split(`hello\ world foo`)
		require.NoError(t, err)
		require.Equal(t, []string{"hello world", "foo"}, result)
	})

	t.Run("mismatched quotes", func(t *testing.T) {
		_, err := shell.Split(`"hello world`)
		require.Error(t, err)
		require.ErrorIs(t, err, shell.ErrMismatchedQuotes)
	})

	t.Run("trailing backslash", func(t *testing.T) {
		_, err := shell.Split(`hello\`)
		require.Error(t, err)
		require.ErrorIs(t, err, shell.ErrTrailingBackslash)
	})

	t.Run("multiple spaces", func(t *testing.T) {
		result, err := shell.Split("a  b   c")
		require.NoError(t, err)
		// Multiple spaces produce empty segments
		require.Contains(t, result, "a")
		require.Contains(t, result, "b")
		require.Contains(t, result, "c")
	})

	t.Run("mixed quotes", func(t *testing.T) {
		result, err := shell.Split(`"hello" 'world'`)
		require.NoError(t, err)
		require.Equal(t, []string{"hello", "world"}, result)
	})
}
