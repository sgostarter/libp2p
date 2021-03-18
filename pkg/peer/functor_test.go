package peer

import "testing"

type TestObjs struct {
	vs []string
}

func TestFunctor(t *testing.T) {
	var tb TestObjs
	//
	tb.vs = []string{"1", "2", "3"}
	fn := func() {
		t.Log(len(tb.vs))
		for i, v := range tb.vs {
			t.Log(i, v)
		}
	}
	tb.vs = nil
	fn()

	//
	tb.vs = []string{"1", "2", "3"}
	vs := tb.vs
	fn = func() {
		t.Log(len(vs))
		for i, v := range vs {
			t.Log(i, v)
		}
	}
	tb.vs = nil
	fn()
}
