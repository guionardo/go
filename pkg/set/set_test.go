package set_test

import (
	"testing"

	"github.com/guionardo/go/pkg/set"
	"github.com/stretchr/testify/assert"
)

func TestSet_Set(t *testing.T) { //nolint:funlen
	t.Parallel()
	t.Run("create_new_should_be_empty", func(t *testing.T) {
		t.Parallel()

		s := set.New[int]()
		assert.Empty(t, s)
	})
	t.Run("create_new_with_values_should_have_correct_length", func(t *testing.T) {
		t.Parallel()

		s := set.New(1, 2, 3)
		assert.Len(t, s, 3)
	})

	t.Run("create_union_from_two_sets_should_have_correct_values", func(t *testing.T) {
		t.Parallel()

		s1 := set.New(1, 2, 3)
		s2 := set.New(3, 4, 5)
		union := s1.Union(s2)
		assert.Len(t, union, 5)
		assert.True(t, union.HasAll(1, 2, 3, 4, 5))
	})

	t.Run("get_diff_from_two_sets", func(t *testing.T) {
		t.Parallel()

		s1 := set.New(1, 2, 3)
		s2 := set.New(3, 4, 5)
		diff := s1.Diff(s2)
		assert.True(t, diff.HasAll(1, 2, 4, 5))
	})

	t.Run("compare_two_equals_sets_should_return_true", func(t *testing.T) {
		t.Parallel()

		s1 := set.New(1, 2, 3)
		s2 := set.New(1, 2, 3)
		assert.True(t, s1.Equals(s2))
	})
	t.Run("compare_two_different_sets_should_return_false", func(t *testing.T) {
		t.Parallel()

		s1 := set.New(1, 2, 3)
		s2 := set.New(1, 2)
		assert.False(t, s1.Equals(s2))

		s2.Add(4)
		assert.False(t, s1.Equals(s2))
	})

	t.Run("iterate_by_set_values", func(t *testing.T) {
		t.Parallel()

		s1 := set.New(1, 2, 3)

		c := 0
		for v := range s1.Iter() {
			c += v
		}

		assert.Equal(t, 6, c)

		for v := range s1.Iter() {
			c = v
			break
		}

		assert.Positive(t, c)
	})

	t.Run("get_intersection_between_sets", func(t *testing.T) {
		t.Parallel()

		s1 := set.New(1, 2, 3)
		s2 := set.New(2, 3, 4)
		inter := s1.Intersection(s2)
		assert.True(t, inter.HasAll(2, 3))
	})

	t.Run("get_filtered_values", func(t *testing.T) {
		t.Parallel()

		s1 := set.New("A", "B", "C", "D", "E")

		var filtered []string
		for v := range s1.Filter(func(s string) bool {
			return s == "A" || s == "E"
		}) {
			filtered = append(filtered, v)
		}

		// For coverege on break
		for range s1.Filter(func(string) bool { return true }) {
			break
		}

		assert.ElementsMatch(t, []string{"A", "E"}, filtered)
	})

	t.Run("test_has_all_not_found_value", func(t *testing.T) {
		t.Parallel()

		s := set.New(1, 2, 3)
		assert.False(t, s.HasAll(1, 2, 4))
	})

	t.Run("test_add_remove_value", func(t *testing.T) {
		t.Parallel()

		s := set.New(1, 2, 3)
		assert.True(t, s.HasAll(1, 2, 3))

		s.Remove(2)
		assert.False(t, s.Has(2))
	})
}
