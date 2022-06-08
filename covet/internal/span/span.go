package span

import (
	"fmt"

	"github.com/johnstarich/go/covet/internal/minmax"
)

type Span struct {
	Start int64 // inclusive
	End   int64 // exclusive
}

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

func (s Span) Len() int64 {
	return s.End - s.Start
}

func (s Span) String() string {
	return fmt.Sprintf("[%d,%d)", s.Start, s.End)
}

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
