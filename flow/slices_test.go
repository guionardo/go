package flow_test

import (
	"testing"

	"github.com/guionardo/go/flow"
	"github.com/stretchr/testify/assert"
)

func TestSliceFirstOrDefault(t *testing.T) {
	t.Parallel()

	t.Run("returns_first_when_not_empty", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, 1, flow.SliceFirstOrDefault([]int{1, 2, 3}, 0))
	})

	t.Run("returns_default_when_empty", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, 42, flow.SliceFirstOrDefault([]int{}, 42))
	})
}

func TestSliceLastOrDefault(t *testing.T) {
	t.Parallel()

	t.Run("returns_last_when_not_empty", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, 3, flow.SliceLastOrDefault([]int{1, 2, 3}, 0))
	})

	t.Run("returns_default_when_empty", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "42", flow.SliceLastOrDefault([]string{}, "42"))
	})
}

func TestRemoveItem(t *testing.T) {
	t.Parallel()

	t.Run("removes_middle_item", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, []int{1, 3}, flow.RemoveItem([]int{1, 2, 3}, 1))
	})

	t.Run("removes_last_item", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, []int{1, 2}, flow.RemoveItem([]int{1, 2, 3}, 2))
	})

	t.Run("removes_first_item", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, []int{2, 3}, flow.RemoveItem([]int{1, 2, 3}, 0))
	})

	t.Run("negative_index_returns_original", func(t *testing.T) {
		t.Parallel()

		original := []int{1, 2, 3}
		assert.Equal(t, original, flow.RemoveItem(original, -1))
	})

	t.Run("out_of_range_index_returns_original", func(t *testing.T) {
		t.Parallel()

		original := []int{1, 2, 3}
		assert.Equal(t, original, flow.RemoveItem(original, 5))
	})

	t.Run("empty_slice_returns_empty", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, []int{}, flow.RemoveItem([]int{}, 0))
	})
}
