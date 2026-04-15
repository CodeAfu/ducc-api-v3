package image

import "errors"

var ErrDuplicateImage = errors.New("Duplicate image detected")
var ErrProtectedImage = errors.New("Image is protected")
var ErrDoesNotExist = errors.New("Image not found")
var ErrInvalidImage = errors.New("Invalid image")
