// Package span contains Span, a numeric range.
package span

import (
	"fmt"

	"github.com/johnstarich/go/covet/internal/minmax"
)

// Span is a numeric range with an inclusive Start and exclusive End index. i.e. [Start, End)
type Span struct {
	Start int64 // inclusive
	End   int64 // exclusive
}

// Intersection returns a Span representing the intersection of s and other.
// Returns false if they do not intersect.
func (s Span) Intersection(other Span) (Span, bool) {
	intersection := Span{
		Start: minmax.MaxInt64(s.Start, other.Start),
		End:   minmax.MinInt64(s.End, other.End),
	}
	if intersection.Start < intersection.End {
		return intersection, true
	}
	return Span{}, false
}

// Len returns the distance between Start and End
func (s Span) Len() int64 {
	return s.End - s.Start
}

func (s Span) String() string {
	return fmt.Sprintf("[%d,%d)", s.Start, s.End)
}

// Merge attempts to combine s and other into a single, unified Span.
// Returns false if they do not intersect and cannot be merged.
func (s Span) Merge(other Span) (Span, bool) {
	_, intersects := s.Intersection(other)
	if !intersects {
		return Span{}, false
	}
	return Span{
		Start: minmax.MinInt64(s.Start, other.Start),
		End:   minmax.MaxInt64(s.End, other.End),
	}, true
}
