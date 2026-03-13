package bingo

import "time"

type BingoResponse struct {
	ID             int64     `json:"id"`
	Title          string    `json:"title"`
	Description    *string   `json:"description"`
	Cells          any       `json:"cells"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	CreatedByEmail *string   `json:"created_by_email"`
}

type CreateBingoRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description"`
	Cells       any     `json:"cells"`
}
