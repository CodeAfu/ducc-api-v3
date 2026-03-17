package bingo

import (
	"errors"
)

var ErrCellsNotJson = errors.New("cells not json")
var ErrCellLenMismatch = errors.New("cells length incorrect (25)")
var ErrInvalidCellKey = errors.New("invalid cell key")
var ErrValueNotString = errors.New("cell value not string")
var ErrBingoNotFound = errors.New("bingo card not found")
