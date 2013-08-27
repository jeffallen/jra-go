package trie

import (
	"errors"
)

const trieSize = 100
const alphaSize = 30

type node struct {
	str  string
	term bool
	key  [alphaSize]uint16 // pointers into the node array
}

type trie [trieSize]node

var NotFound = errors.New("key not found")
var BadKey = errors.New("key has characters out of range")

// add expects keys to be in sorted order
/*
func (t *trie)add(keys [][]byte) {
	next := uint16(1)
	for _, key := range keys {
		cur := uint16(1)
		i := 0
		for cur != 0 {
			k, err := keyToIndex(key[i])
			if err != nil {
				panic("bad character in key")
			}
	}
}
*/

func (t *trie) find(key []byte) (string, error) {
	cur := uint16(1)
	i := 0
	for cur != 0 {
		if i == len(key) || t[cur].term {
			return t[cur].str, nil
		}
		k, err := keyToIndex(key[i])
		if err != nil {
			return "", err
		}
		i++
		cur = t[cur].key[k]
	}
	return "", NotFound
}

func keyToIndex(in byte) (uint8, error) {
	var out int = int(in)
	if out == '-' {
		out = '@'
	}
	if out >= 'a' {
		out -= 32 // to lower
	}
	out -= '@' // @=0, A=1 ...
	if out < 0 || out > alphaSize-1 {
		return 0, BadKey
	}
	return uint8(out), nil
}
