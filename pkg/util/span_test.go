package util

import (
	"github.com/pingcap/check"
	"github.com/pingcap/tidb/tablecodec"
	"github.com/pingcap/tidb/util/codec"
)

type spanSuite struct{}

var _ = check.Suite(&spanSuite{})

func (s *spanSuite) TestStartCompare(c *check.C) {
	tests := []struct {
		lhs []byte
		rhs []byte
		res int
	}{
		{nil, nil, 0},
		{nil, []byte{}, -1},
		{[]byte{}, nil, 1},
		{[]byte{}, []byte{}, 0},
		{[]byte{1}, []byte{2}, -1},
		{[]byte{2}, []byte{1}, 1},
		{[]byte{3}, []byte{3}, 0},
	}

	for _, t := range tests {
		c.Assert(StartCompare(t.lhs, t.rhs), check.Equals, t.res)
	}
}

func (s *spanSuite) TestEndCompare(c *check.C) {
	tests := []struct {
		lhs []byte
		rhs []byte
		res int
	}{
		{nil, nil, 0},
		{nil, []byte{}, 1},
		{[]byte{}, nil, -1},
		{[]byte{}, []byte{}, 0},
		{[]byte{1}, []byte{2}, -1},
		{[]byte{2}, []byte{1}, 1},
		{[]byte{3}, []byte{3}, 0},
	}

	for _, t := range tests {
		c.Assert(EndCompare(t.lhs, t.rhs), check.Equals, t.res)
	}
}

func (s *spanSuite) TestIntersect(c *check.C) {
	tests := []struct {
		lhs Span
		rhs Span
		// Set nil for non-intersect
		res *Span
	}{
		{Span{nil, []byte{1}}, Span{[]byte{1}, nil}, nil},
		{Span{nil, nil}, Span{nil, nil}, &Span{nil, nil}},
		{Span{nil, nil}, Span{[]byte{1}, []byte{2}}, &Span{[]byte{1}, []byte{2}}},
		{Span{[]byte{0}, []byte{3}}, Span{[]byte{1}, []byte{2}}, &Span{[]byte{1}, []byte{2}}},
		{Span{[]byte{0}, []byte{2}}, Span{[]byte{1}, []byte{2}}, &Span{[]byte{1}, []byte{2}}},
	}

	for _, t := range tests {
		c.Log("running..", t)
		res, err := Intersect(t.lhs, t.rhs)
		if t.res == nil {
			c.Assert(err, check.NotNil)
		} else {
			c.Assert(res, check.DeepEquals, *t.res)
		}

		// Swap lhs and rhs, should get the same result
		res2, err2 := Intersect(t.rhs, t.lhs)
		if t.res == nil {
			c.Assert(err2, check.NotNil)
		} else {
			c.Assert(res2, check.DeepEquals, *t.res)
		}
	}
}

func (s *spanSuite) TestGetTableSpan(c *check.C) {
	span := GetTableSpan(123, true)
	c.Assert(span.Start, check.Less, span.End)
	prefix := []byte(tablecodec.GenTablePrefix(123))
	c.Assert(span.Start, check.Greater, codec.EncodeBytes(nil, prefix))
	prefix[len(prefix)-1]++
	c.Assert(span.End, check.Less, codec.EncodeBytes(nil, prefix))

	span = GetTableSpan(123, false)
	c.Assert(span.Start, check.Less, span.End)
	prefix = []byte(tablecodec.GenTablePrefix(123))
	c.Assert(span.Start, check.Greater, prefix)
	prefix[len(prefix)-1]++
	c.Assert(span.End, check.Less, prefix)
}

func (s *spanSuite) TestSpanHack(c *check.C) {
	testCases := []struct {
		input  Span
		expect Span
	}{
		{Span{nil, nil}, Span{[]byte{}, UpperBoundKey}},
		{Span{nil, []byte{1}}, Span{[]byte{}, []byte{1}}},
		{Span{[]byte{1}, nil}, Span{[]byte{1}, UpperBoundKey}},
		{Span{[]byte{1}, []byte{2}}, Span{[]byte{1}, []byte{2}}},
		{Span{[]byte{}, []byte{}}, Span{[]byte{}, []byte{}}},
	}

	for _, tc := range testCases {
		c.Assert(tc.input.Hack(), check.DeepEquals, tc.expect)
	}
}