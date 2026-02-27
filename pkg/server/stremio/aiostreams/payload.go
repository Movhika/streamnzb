package aiostreams

// Payload matches the JSON shape AIOStreams sends in the config query param
// (subset of UserDataSchema used for filtering/sorting).
type Payload struct {
	SortCriteria struct {
		Global []SortCriterion `json:"global"`
	} `json:"sortCriteria"`

	PreferredResolutions []string `json:"preferredResolutions"`
	ExcludedResolutions  []string `json:"excludedResolutions"`
	IncludedResolutions  []string `json:"includedResolutions"`
	RequiredResolutions  []string `json:"requiredResolutions"`

	ExcludedQualities  []string `json:"excludedQualities"`
	PreferredQualities []string `json:"preferredQualities"`
	IncludedQualities  []string `json:"includedQualities"`
	RequiredQualities  []string `json:"requiredQualities"`

	ExcludedVisualTags  []string `json:"excludedVisualTags"`
	PreferredVisualTags []string `json:"preferredVisualTags"`
	IncludedVisualTags  []string `json:"includedVisualTags"`
	RequiredVisualTags  []string `json:"requiredVisualTags"`

	ExcludedAudioTags  []string `json:"excludedAudioTags"`
	PreferredAudioTags []string `json:"preferredAudioTags"`
	IncludedAudioTags  []string `json:"includedAudioTags"`
	RequiredAudioTags  []string `json:"requiredAudioTags"`

	ExcludedAudioChannels  []string `json:"excludedAudioChannels"`
	PreferredAudioChannels []string `json:"preferredAudioChannels"`
	IncludedAudioChannels  []string `json:"includedAudioChannels"`
	RequiredAudioChannels  []string `json:"requiredAudioChannels"`

	ExcludedStreamTypes  []string `json:"excludedStreamTypes"`
	PreferredStreamTypes []string `json:"preferredStreamTypes"`
	IncludedStreamTypes  []string `json:"includedStreamTypes"`
	RequiredStreamTypes  []string `json:"requiredStreamTypes"`

	ExcludedEncodes  []string `json:"excludedEncodes"`
	PreferredEncodes []string `json:"preferredEncodes"`
	IncludedEncodes  []string `json:"includedEncodes"`
	RequiredEncodes  []string `json:"requiredEncodes"`

	ExcludedLanguages  []string `json:"excludedLanguages"`
	PreferredLanguages []string `json:"preferredLanguages"`
	IncludedLanguages  []string `json:"includedLanguages"`
	RequiredLanguages  []string `json:"requiredLanguages"`

	ExcludedReleaseGroups  []string `json:"excludedReleaseGroups"`
	PreferredReleaseGroups []string `json:"preferredReleaseGroups"`
	IncludedReleaseGroups  []string `json:"includedReleaseGroups"`
	RequiredReleaseGroups  []string `json:"requiredReleaseGroups"`
}

// SortCriterion is one entry in sortCriteria.global.
type SortCriterion struct {
	Key       string `json:"key"`
	Direction string `json:"direction"`
}
