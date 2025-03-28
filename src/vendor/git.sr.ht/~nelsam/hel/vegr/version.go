package vegr

// EnforceVersion is used by generated code to ensure that runtime code supports
// the version of the generated code. It relies on uint underflow to break if
// the version of hel that it was generated with is not supported by the
// imported version of this library.
//
// The runtime can be updated with:
//
//	go get -u git.sr.ht/~nelsam/hel
//
// The generated code can be updated with:
//
//	hel
//
// Credit goes to github.com/protocol-buffers/protobuf-go for this idea.
type EnforceVersion uint

const (
	// MinVersion is the minimum version of hel that this library can work with.
	// Any version below v0.{MinVersion}.0 uses legacy code that is no longer
	// supported in this version of vegr.
	MinVersion = 7

	// MaxVersion is the maximum version of hel that this library can work with.
	// Any version above v0.{MaxVersion}.* may rely on new features in vegr that
	// don't exist in this version.
	MaxVersion = Version

	// Version is the current version of hel.
	Version = 7
)
