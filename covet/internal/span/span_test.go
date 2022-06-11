package span

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntersect(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a, b         Span
		intersection Span
	}{
		{ // same span
			a:            Span{Start: 1, End: 2},
			b:            Span{Start: 1, End: 2},
			intersection: Span{Start: 1, End: 2},
		},
		{ // adjacent
			a:            Span{Start: 1, End: 2},
			b:            Span{Start: 2, End: 3},
			intersection: Span{},
		},
		{ // disjoint
			a:            Span{Start: 1, End: 2},
			b:            Span{Start: 3, End: 4},
			intersection: Span{},
		},
		{ // leading a, trailing b
			a:            Span{Start: 1, End: 10},
			b:            Span{Start: 5, End: 15},
			intersection: Span{Start: 5, End: 10},
		},
		{ // trailing a, leading b
			a:            Span{Start: 5, End: 15},
			b:            Span{Start: 1, End: 10},
			intersection: Span{Start: 5, End: 10},
		},
		{ // a inside b
			a:            Span{Start: 4, End: 6},
			b:            Span{Start: 1, End: 10},
			intersection: Span{Start: 4, End: 6},
		},
		{ // b inside a
			a:            Span{Start: 1, End: 10},
			b:            Span{Start: 4, End: 6},
			intersection: Span{Start: 4, End: 6},
		},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(fmt.Sprintf("%s∩%s", tc.a, tc.b), func(t *testing.T) {
			t.Parallel()
			span, intersects := tc.a.Intersection(tc.b)
			assert.Equal(t, tc.intersection, span)
			assert.Equal(t, tc.intersection != Span{}, intersects)
		})
	}
}

func TestMerge(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a, b   Span
		merged Span
	}{
		{ // same span
			a:      Span{Start: 1, End: 2},
			b:      Span{Start: 1, End: 2},
			merged: Span{Start: 1, End: 2},
		},
		{ // adjacent
			a:      Span{Start: 1, End: 2},
			b:      Span{Start: 2, End: 3},
			merged: Span{},
		},
		{ // disjoint
			a:      Span{Start: 1, End: 2},
			b:      Span{Start: 3, End: 4},
			merged: Span{},
		},
		{ // leading a, trailing b
			a:      Span{Start: 1, End: 10},
			b:      Span{Start: 5, End: 15},
			merged: Span{Start: 1, End: 15},
		},
		{ // trailing a, leading b
			a:      Span{Start: 5, End: 15},
			b:      Span{Start: 1, End: 10},
			merged: Span{Start: 1, End: 15},
		},
		{ // a inside b
			a:      Span{Start: 4, End: 6},
			b:      Span{Start: 1, End: 10},
			merged: Span{Start: 1, End: 10},
		},
		{ // b inside a
			a:      Span{Start: 1, End: 10},
			b:      Span{Start: 4, End: 6},
			merged: Span{Start: 1, End: 10},
		},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(fmt.Sprintf("%s∩%s", tc.a, tc.b), func(t *testing.T) {
			t.Parallel()
			span, merged := tc.a.Merge(tc.b)
			assert.Equal(t, tc.merged, span)
			assert.Equal(t, tc.merged != Span{}, merged)
		})
	}
}
