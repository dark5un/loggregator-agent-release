package vegr

import "git.sr.ht/~nelsam/hel/vegr/ret"

// BlockChan initializes a channel for use when a method may want to block a
// return.
func BlockChan() ret.Blocker {
	ch := make(ret.Blocker, 1)
	ch <- struct{}{}
	return ch
}
