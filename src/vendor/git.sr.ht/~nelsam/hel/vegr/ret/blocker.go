package ret

// Blocker is a special type of signal channel used to block method calls from
// returning.
type Blocker chan struct{}
