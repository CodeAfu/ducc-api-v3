package genshin

import (
	"errors"
)

var (
	ErrCharAlreadyExists   = errors.New("Character already exists")
	ErrCharDoesNotExist    = errors.New("Character does not exist")
	ErrProfileDoesNotExist = errors.New("Profile does not exist")
	ErrIconUrlNotFound     = errors.New("Icon url not found")
	ErrInvalidElement      = errors.New("Invalid element name")
	ErrInvalidStars        = errors.New("Invalid stars value")
	ErrExternalAPINotOK    = errors.New("External api returned non ok status")
)
