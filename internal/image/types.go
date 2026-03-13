package image

import "time"

type ImageResponse struct {
	ID          int64     `json:"id"`
	ImgData     string    `json:"img_data"`
	ImgHash     string    `json:"img_hash"`
	AddedBy     string    `json:"added_by"`
	Filename    string    `json:"filename"`
	Fileext     string    `json:"fileext"`
	IsProtected bool      `json:"is_protected"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateImageRequest struct {
	ImgData     string `json:"img_data"`
	Filename    string `json:"filename"`
	Fileext     string `json:"fileext"`
	IsProtected bool   `json:"is_protected"`
}
