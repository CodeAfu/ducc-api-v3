package cache

import "fmt"

const (
	GenshinCharsAllKey    = "genshin:characters:all"
	GenshinProfilesAllKey = "genshin:profiles:all"
	GenshinElementsAllKey = "genshin:elements:all"
)

func GenshinCharKey(id int64) string {
	return fmt.Sprintf("genshin:characters:%d", id)
}

func GenshinProfKey(id int64) string {
	return fmt.Sprintf("genshin:profiles:%d", id)
}

func GenshinProfCharsKey(profId int64) string {
	return fmt.Sprintf("genshin:profiles:%d:characters:all", profId)
}

func GenshinProfStatsKey(profId int64) string {
	return fmt.Sprintf("genshin:profiles:%d:stats", profId)
}

func GenshinElementIconKey(elementName string) string {
	return fmt.Sprintf("genshin:elements:%s:icon", elementName)
}
