package image

import "errors"

var ErrDuplicateImage = errors.New("duplicate image")
var ErrProtectedImage = errors.New("image is protected")
