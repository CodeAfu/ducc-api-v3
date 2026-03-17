package genshin

import "github.com/jackc/pgx/v5/pgtype"

type charResponse struct {
	ID          int64       `json:"id"`
	Name        string      `json:"name"`
	ElementName string      `json:"element_name"`
	Stars       int16       `json:"stars"`
	Icon        []byte      `json:"icon"`
	Notes       pgtype.Text `json:"notes"`
}

type addCharRequest struct {
	Name        string      `json:"name"`
	ElementName string      `json:"element_name"`
	Stars       int16       `json:"stars"`
	Icon        []byte      `json:"icon"`
	Notes       pgtype.Text `json:"notes"`
}

type editCharRequest struct {
	Name        string      `json:"name"`
	ElementName string      `json:"element_name"`
	Stars       int16       `json:"stars"`
	Icon        []byte      `json:"icon"`
	Notes       pgtype.Text `json:"notes"`
}
