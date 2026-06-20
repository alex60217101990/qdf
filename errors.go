package qdf

import (
	"errors"
	"fmt"
)

var (
	ErrShortBuffer    = errors.New("qdf: short buffer")
	ErrBadMagic       = errors.New("qdf: bad magic / not a qdf stream")
	ErrBadVersion     = errors.New("qdf: unsupported wire version")
	ErrBadTag         = errors.New("qdf: unknown tag")
	ErrTypeMismatch   = errors.New("qdf: type mismatch on decode")
	ErrInvalidLength  = errors.New("qdf: invalid length prefix")
	ErrUnknownStateID = errors.New("qdf: unknown state-table id")
	ErrUnsupported    = errors.New("qdf: unsupported type")
	ErrCycleDetected  = errors.New("qdf: pointer cycle detected (max depth exceeded)")
	ErrFieldNotFound  = errors.New("qdf: query predicate field not found")
	// ErrStreamBadFlags is returned by StreamDecoder.Decode when the stream
	// header carries whole-payload flags that do not apply to a frame stream
	// (FlagRANS / FlagColIndex). A conforming StreamEncoder never sets them; a
	// hostile stream claiming them would, if honored, swap the shared window
	// buffer mid-stream and desync framing. The decoder is latched broken.
	ErrStreamBadFlags = errors.New("qdf: stream header has unsupported whole-payload flags")
)

// QueryError describes why a filtering/projecting decode (Unmarshal with
// QueryOptions) could not proceed. It wraps one of ErrUnsupported,
// ErrTypeMismatch, or ErrFieldNotFound, so callers can categorise the failure
// with errors.Is and read the specifics with errors.As.
type QueryError struct {
	Err   error   // wrapped sentinel
	Op    string  // e.g. "predicate pushdown"
	Field string  // filter/projection field involved, if any
	Want  colKind // kind implied by the predicate's T (kind mismatches)
	Got   colKind // kind found on the wire (kind mismatches)
}

func (e *QueryError) Error() string {
	switch {
	case e.Field != "" && errors.Is(e.Err, ErrTypeMismatch):
		return fmt.Sprintf("qdf: %s: field %q kind %s does not match wire kind %s: %v",
			e.Op, e.Field, e.Want, e.Got, e.Err)
	case e.Field != "":
		return fmt.Sprintf("qdf: %s: field %q: %v", e.Op, e.Field, e.Err)
	default:
		return fmt.Sprintf("qdf: %s: %v", e.Op, e.Err)
	}
}

func (e *QueryError) Unwrap() error { return e.Err }
