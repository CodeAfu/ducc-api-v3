package image

import "errors"

var ErrDuplicateImage = errors.New("duplicate image detected")
var ErrProtectedImage = errors.New("image is protected")
var ErrDoesNotExist = errors.New("not found")
var ErrInvalidImage = errors.New("invalid image")
