package genshin

type elementStat struct {
	ElementName string `json:"element_name"`
	CharCount   int32  `json:"count"`
}

type profileStatsResponse struct {
	CharCount     int32         `json:"char_count"`
	ElementCounts []elementStat `json:"element_counts"`
}

type editGenshinProfileRequest struct {
	ID    int64   `json:"id"`
	Name  *string `json:"name"`
	Notes *string `json:"notes"`
}

type ProfileResponse struct {
	ID         int64               `json:"id"`
	Name       string              `json:"name"`
	Notes      string              `json:"notes"`
	Characters []CharacterResponse `json:"characters"`
}

type CharacterResponse struct {
	CharID          int64  `json:"char_id"`
	Name            string `json:"name"`
	DisplayName     string `json:"display_name"`
	Level           int16  `json:"level"`
	AscLevel        int16  `json:"asc_level"`
	Stars           int16  `json:"stars"`
	Constellation   int16  `json:"constellation"`
	TalentNa        int16  `json:"talent_na"`
	TalentE         int16  `json:"talent_e"`
	TalentQ         int16  `json:"talent_q"`
	TalentNaBoosted bool   `json:"talent_na_boosted"`
	TalentEBoosted  bool   `json:"talent_e_boosted"`
	TalentQBoosted  bool   `json:"talent_q_boosted"`
	CharNotes       string `json:"char_notes"`
	ElementName     string `json:"element_name"`
	ElementIcon     string `json:"element_icon"`
}
