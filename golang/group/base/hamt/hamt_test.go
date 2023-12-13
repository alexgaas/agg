package hamt

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHamt(t *testing.T) {
	root := Hamt{children: make([]*Hamt, 2), key: "", value: 0}
	a := Hamt{children: make([]*Hamt, 2), key: "a", value: 1}
	b := Hamt{children: make([]*Hamt, 2), key: "b", value: 2}
	c := Hamt{children: make([]*Hamt, 2), key: "c", value: 3}

	hamt_add(&root, &a)
	hamt_add(&root, &b)
	hamt_add(&root, &c)

	require.True(t, hamt_find(&root, "a").value == 1)
	require.True(t, hamt_find(&root, "b").value == 2)
	require.True(t, hamt_find(&root, "c").value == 3)
}
