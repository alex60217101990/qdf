package qdf

import "errors"

var (
	ErrShortBuffer    = errors.New("qdf: short buffer")
	ErrBadMagic       = errors.New("qdf: bad magic / not a qdf stream")
	ErrBadVersion     = errors.New("qdf: unsupported wire version")
	ErrBadTag         = errors.New("qdf: unknown tag")
	ErrTypeMismatch   = errors.New("qdf: type mismatch on decode")
	ErrInvalidLength  = errors.New("qdf: invalid length prefix")
	ErrUnknownStateID = errors.New("qdf: unknown state-table id")
	ErrUnsupported    = errors.New("qdf: unsupported type")
)
