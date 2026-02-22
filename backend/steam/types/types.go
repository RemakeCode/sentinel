package types

// Language represents a language from Steam
type Language struct {
	DisplayName string `json:"displayName"`
	API         string `json:"api"`
	WebAPI      string `json:"webapi"`
}

// SteamLanguages is the list of available Steam languages
var SteamLanguages = []Language{
	{DisplayName: "Arabic", API: "arabic", WebAPI: "ar"},
	{DisplayName: "Bulgarian", API: "bulgarian", WebAPI: "bg"},
	{DisplayName: "Simplified Chinese", API: "schinese", WebAPI: "zh-CN"},
	{DisplayName: "Traditional Chinese", API: "tchinese", WebAPI: "zh-TW"},
	{DisplayName: "Czech", API: "czech", WebAPI: "cs"},
	{DisplayName: "Danish", API: "danish", WebAPI: "da"},
	{DisplayName: "Dutch", API: "dutch", WebAPI: "nl"},
	{DisplayName: "English", API: "english", WebAPI: "en"},
	{DisplayName: "Finnish", API: "finnish", WebAPI: "fi"},
	{DisplayName: "French", API: "french", WebAPI: "fr"},
	{DisplayName: "German", API: "german", WebAPI: "de"},
	{DisplayName: "Greek", API: "greek", WebAPI: "el"},
	{DisplayName: "Hungarian", API: "hungarian", WebAPI: "hu"},
	{DisplayName: "Italian", API: "italian", WebAPI: "it"},
	{DisplayName: "Japanese", API: "japanese", WebAPI: "ja"},
	{DisplayName: "Korean", API: "koreana", WebAPI: "ko"},
	{DisplayName: "Norwegian", API: "norwegian", WebAPI: "no"},
	{DisplayName: "Polish", API: "polish", WebAPI: "pl"},
	{DisplayName: "Portuguese", API: "portuguese", WebAPI: "pt"},
	{DisplayName: "Portuguese - Brazil", API: "brazilian", WebAPI: "pt-BR"},
	{DisplayName: "Romanian", API: "romanian", WebAPI: "ro"},
	{DisplayName: "Russian", API: "russian", WebAPI: "ru"},
	{DisplayName: "Spanish - Spain", API: "spanish", WebAPI: "es"},
	{DisplayName: "Spanish - Latin America", API: "latam", WebAPI: "es-419"},
	{DisplayName: "Swedish", API: "swedish", WebAPI: "sv"},
	{DisplayName: "Thai", API: "thai", WebAPI: "th"},
	{DisplayName: "Turkish", API: "turkish", WebAPI: "tr"},
	{DisplayName: "Ukrainian", API: "ukrainian", WebAPI: "uk"},
	{DisplayName: "Vietnamese", API: "vietnamese", WebAPI: "vn"},
}

// GetSteamLanguages returns the list of available Steam languages
func GetSteamLanguages() []Language {
	return SteamLanguages
}
