package genshin

import (
	"errors"
)

var (
	ErrCharAlreadyExists   = errors.New("character already exists")
	ErrCharDoesNotExist    = errors.New("character does not exist")
	ErrProfileDoesNotExist = errors.New("profile does not exist")
	ErrIconUrlNotFound     = errors.New("icon url not found")
	ErrInvalidElement      = errors.New("invalid element name")
	ErrInvalidStars        = errors.New("invalid stars value")
	ErrExternalAPINotOK    = errors.New("external api returned non ok status")
)
