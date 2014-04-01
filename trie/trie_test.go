package trie

import (
	"testing"
)

type test struct {
	key    []byte
	expect string
	err    error
}

var thetests = []test{
	{[]byte("hello"), "", NotFound},
	{[]byte("a"), "a", nil},
	{[]byte("aa"), "aa", nil},
	{[]byte("abcd"), "abcd", nil},
}

var tr *trie

func init() {
	tr = new(trie)
	tr[1].key[1] = 2
	tr[2].str = "a"
	tr[2].key[1] = 3
	tr[2].key[2] = 4
	tr[3].str = "aa"
	tr[4].str = "abcd"
	tr[4].term = true
}

func TestFind(t *testing.T) {
	for i, x := range thetests {
		s, err := tr.find(x.key)
		if x.err != err {
			t.Fatal("test", i, "expected err:", x.err, ", got err", err)
		}
		if x.expect != s {
			t.Fatal("test", i, "expected result:", x.expect, ", got string", s)
		}
	}
}

// ex: ts=2
