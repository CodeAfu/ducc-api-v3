package genshin

import (
	"errors"
)

var (
	ErrCharAlreadyExists = errors.New("character already exists")
	ErrCharDoesNotExist  = errors.New("character does not exist")
	ErrInvalidElement    = errors.New("invalid element name")
	ErrInvalidStars      = errors.New("invalid stars value")
)
