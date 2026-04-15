package bingo

import (
	"errors"
)

var ErrCellsNotJson = errors.New("Cells not in JSON format")
var ErrCellLenMismatch = errors.New("Cells array length is incorrect (max=25)")
var ErrInvalidCellKey = errors.New("Invalid cell key")
var ErrValueNotString = errors.New("Cell value is not of type string")
var ErrBingoNotFound = errors.New("Bingo Card not found")
