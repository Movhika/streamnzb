package config

import (
	"encoding/json"
	"sort"

	"streamnzb/pkg/core/config/pttoptions"
)

type FilterConfig struct {
	AudioIncluded      []string `json:"audio_included,omitempty"`
	AudioRequired      []string `json:"audio_required"`
	AudioExcluded      []string `json:"audio_excluded"`
	BitDepthIncluded   []string `json:"bit_depth_included,omitempty"`
	BitDepthRequired   []string `json:"bit_depth_required"`
	BitDepthExcluded   []string `json:"bit_depth_excluded"`
	ChannelsIncluded   []string `json:"channels_included,omitempty"`
	ChannelsRequired   []string `json:"channels_required"`
	ChannelsExcluded   []string `json:"channels_excluded"`
	CodecIncluded      []string `json:"codec_included,omitempty"`
	CodecRequired      []string `json:"codec_required"`
	CodecExcluded      []string `json:"codec_excluded"`
	ContainerIncluded  []string `json:"container_included,omitempty"`
	ContainerRequired  []string `json:"container_required"`
	ContainerExcluded  []string `json:"container_excluded"`
	EditionIncluded    []string `json:"edition_included,omitempty"`
	EditionRequired    []string `json:"edition_required"`
	EditionExcluded    []string `json:"edition_excluded"`
	HDRIncluded        []string `json:"hdr_included,omitempty"`
	HDRRequired        []string `json:"hdr_required"`
	HDRExcluded        []string `json:"hdr_excluded"`
	LanguagesIncluded  []string `json:"languages_included,omitempty"`
	LanguagesRequired  []string `json:"languages_required"`
	LanguagesExcluded  []string `json:"languages_excluded"`
	NetworkIncluded    []string `json:"network_included,omitempty"`
	NetworkRequired    []string `json:"network_required"`
	NetworkExcluded    []string `json:"network_excluded"`
	QualityIncluded    []string `json:"quality_included,omitempty"`
	QualityRequired    []string `json:"quality_required"`
	QualityExcluded    []string `json:"quality_excluded"`
	RegionIncluded     []string `json:"region_included,omitempty"`
	RegionRequired     []string `json:"region_required"`
	RegionExcluded     []string `json:"region_excluded"`
	ResolutionIncluded []string `json:"resolution_included,omitempty"`
	ResolutionRequired []string `json:"resolution_required"`
	ResolutionExcluded []string `json:"resolution_excluded"`
	ThreeDIncluded     []string `json:"three_d_included,omitempty"`
	ThreeDRequired     []string `json:"three_d_required"`
	ThreeDExcluded     []string `json:"three_d_excluded"`
	GroupIncluded      []string `json:"group_included,omitempty"`
	GroupRequired      []string `json:"group_required"`
	GroupExcluded      []string `json:"group_excluded"`

	DubbedExcluded    *bool `json:"dubbed_excluded,omitempty"`
	HardcodedExcluded *bool `json:"hardcoded_excluded,omitempty"`
	ProperRequired    *bool `json:"proper_required,omitempty"`
	RepackRequired    *bool `json:"repack_required,omitempty"`
	RepackExcluded    *bool `json:"repack_excluded,omitempty"`
	ExtendedRequired  *bool `json:"extended_required,omitempty"`
	UnratedRequired   *bool `json:"unrated_required,omitempty"`

	MinSizeGB float64 `json:"min_size_gb"`
	MaxSizeGB float64 `json:"max_size_gb"`
	MinYear   int     `json:"min_year"`
	MaxYear   int     `json:"max_year"`

	MinAgeHours float64 `json:"min_age_hours"`
	MaxAgeHours float64 `json:"max_age_hours"`

	KeywordsExcluded []string `json:"keywords_excluded,omitempty"`
	KeywordsRequired []string `json:"keywords_required,omitempty"`

	RegexExcluded []string `json:"regex_excluded,omitempty"`
	RegexRequired []string `json:"regex_required,omitempty"`

	AvailNZBIncluded []string `json:"availnzb_included,omitempty"`
	AvailNZBRequired []string `json:"availnzb_required,omitempty"`
	AvailNZBExcluded []string `json:"availnzb_excluded,omitempty"`

	SizePerResolution map[string]SizeRange `json:"size_per_resolution,omitempty"`

	MinBitrateKbps float64 `json:"min_bitrate_kbps"`
	MaxBitrateKbps float64 `json:"max_bitrate_kbps"`
}

type SizeRange struct {
	MinGB float64 `json:"min_gb"`
	MaxGB float64 `json:"max_gb"`
}

type SortConfig struct {
	PreferredResolution []string `json:"preferred_resolution"`
	PreferredCodec      []string `json:"preferred_codec"`
	PreferredAudio      []string `json:"preferred_audio"`
	PreferredQuality    []string `json:"preferred_quality"`
	PreferredVisualTag  []string `json:"preferred_visual_tag"`
	PreferredChannels   []string `json:"preferred_channels"`
	PreferredBitDepth   []string `json:"preferred_bit_depth"`
	PreferredContainer  []string `json:"preferred_container"`
	PreferredLanguages  []string `json:"preferred_languages"`
	PreferredGroup      []string `json:"preferred_group"`
	PreferredEdition    []string `json:"preferred_edition"`
	PreferredNetwork    []string `json:"preferred_network"`
	PreferredRegion     []string `json:"preferred_region"`
	PreferredThreeD     []string `json:"preferred_three_d"`
	PreferredAvailNZB   []string `json:"preferred_availnzb,omitempty"`

	GrabWeight float64 `json:"grab_weight"`
	AgeWeight  float64 `json:"age_weight"`

	KeywordsPreferred []string `json:"keywords_preferred,omitempty"`
	KeywordsWeight    float64  `json:"keywords_weight,omitempty"`

	RegexPreferred []string `json:"regex_preferred,omitempty"`
	RegexWeight    float64  `json:"regex_weight,omitempty"`

	AvailNZBWeight float64 `json:"availnzb_weight,omitempty"`

	SortCriteriaOrder []string `json:"sort_criteria_order,omitempty"`

	UseCustomScoring *bool    `json:"use_custom_scoring,omitempty"`
	ResolutionWeight float64  `json:"resolution_weight,omitempty"`
	CodecWeight      float64  `json:"codec_weight,omitempty"`
	AudioWeight      float64  `json:"audio_weight,omitempty"`
	QualityWeight    float64  `json:"quality_weight,omitempty"`
	VisualTagWeight  float64  `json:"visual_tag_weight,omitempty"`
	ChannelsWeight   float64  `json:"channels_weight,omitempty"`
	BitDepthWeight   float64  `json:"bit_depth_weight,omitempty"`
	ContainerWeight  float64  `json:"container_weight,omitempty"`
	LanguagesWeight  float64  `json:"languages_weight,omitempty"`
	GroupWeight      float64  `json:"group_weight,omitempty"`
	EditionWeight    float64  `json:"edition_weight,omitempty"`
	NetworkWeight    float64  `json:"network_weight,omitempty"`
	RegionWeight     float64  `json:"region_weight,omitempty"`
	ThreeDWeight     float64  `json:"three_d_weight,omitempty"`
	GroupOrderTier1  []string `json:"group_order_tier1,omitempty"`
	GroupOrderTier2  []string `json:"group_order_tier2,omitempty"`
	GroupOrderTier3  []string `json:"group_order_tier3,omitempty"`
	GroupTier1Points int      `json:"group_tier1_points,omitempty"`
	GroupTier2Points int      `json:"group_tier2_points,omitempty"`
	GroupTier3Points int      `json:"group_tier3_points,omitempty"`
}

type filterConfigRaw struct {
	AudioIncluded      []string `json:"audio_included"`
	AudioRequired      []string `json:"audio_required"`
	AudioExcluded      []string `json:"audio_excluded"`
	BitDepthIncluded   []string `json:"bit_depth_included"`
	BitDepthRequired   []string `json:"bit_depth_required"`
	BitDepthExcluded   []string `json:"bit_depth_excluded"`
	ChannelsIncluded   []string `json:"channels_included"`
	ChannelsRequired   []string `json:"channels_required"`
	ChannelsExcluded   []string `json:"channels_excluded"`
	CodecIncluded      []string `json:"codec_included"`
	CodecRequired      []string `json:"codec_required"`
	CodecExcluded      []string `json:"codec_excluded"`
	ContainerIncluded  []string `json:"container_included"`
	ContainerRequired  []string `json:"container_required"`
	ContainerExcluded  []string `json:"container_excluded"`
	EditionIncluded    []string `json:"edition_included"`
	EditionRequired    []string `json:"edition_required"`
	EditionExcluded    []string `json:"edition_excluded"`
	HDRIncluded        []string `json:"hdr_included"`
	HDRRequired        []string `json:"hdr_required"`
	HDRExcluded        []string `json:"hdr_excluded"`
	LanguagesIncluded  []string `json:"languages_included"`
	LanguagesRequired  []string `json:"languages_required"`
	LanguagesExcluded  []string `json:"languages_excluded"`
	NetworkIncluded    []string `json:"network_included"`
	NetworkRequired    []string `json:"network_required"`
	NetworkExcluded    []string `json:"network_excluded"`
	QualityIncluded    []string `json:"quality_included"`
	QualityRequired    []string `json:"quality_required"`
	QualityExcluded    []string `json:"quality_excluded"`
	RegionIncluded     []string `json:"region_included"`
	RegionRequired     []string `json:"region_required"`
	RegionExcluded     []string `json:"region_excluded"`
	ResolutionIncluded []string `json:"resolution_included"`
	ResolutionRequired []string `json:"resolution_required"`
	ResolutionExcluded []string `json:"resolution_excluded"`
	ThreeDIncluded     []string `json:"three_d_included"`
	ThreeDRequired     []string `json:"three_d_required"`
	ThreeDExcluded     []string `json:"three_d_excluded"`
	GroupIncluded      []string `json:"group_included"`
	GroupRequired      []string `json:"group_required"`
	GroupExcluded      []string `json:"group_excluded"`

	DubbedExcluded    *bool `json:"dubbed_excluded,omitempty"`
	HardcodedExcluded *bool `json:"hardcoded_excluded,omitempty"`
	ProperRequired    *bool `json:"proper_required,omitempty"`
	RepackRequired    *bool `json:"repack_required,omitempty"`
	RepackExcluded    *bool `json:"repack_excluded,omitempty"`
	ExtendedRequired  *bool `json:"extended_required,omitempty"`
	UnratedRequired   *bool `json:"unrated_required,omitempty"`

	AudioInclude      []string `json:"audio_include"`
	AudioAvoid        []string `json:"audio_avoid"`
	BitDepthInclude   []string `json:"bit_depth_include"`
	BitDepthAvoid     []string `json:"bit_depth_avoid"`
	ChannelsInclude   []string `json:"channels_include"`
	ChannelsAvoid     []string `json:"channels_avoid"`
	CodecInclude      []string `json:"codec_include"`
	CodecAvoid        []string `json:"codec_avoid"`
	ContainerInclude  []string `json:"container_include"`
	ContainerAvoid    []string `json:"container_avoid"`
	EditionInclude    []string `json:"edition_include"`
	EditionAvoid      []string `json:"edition_avoid"`
	HDRInclude        []string `json:"hdr_include"`
	HDRAvoid          []string `json:"hdr_avoid"`
	LanguagesInclude  []string `json:"languages_include"`
	LanguagesAvoid    []string `json:"languages_avoid"`
	NetworkInclude    []string `json:"network_include"`
	NetworkAvoid      []string `json:"network_avoid"`
	QualityInclude    []string `json:"quality_include"`
	QualityAvoid      []string `json:"quality_avoid"`
	RegionInclude     []string `json:"region_include"`
	RegionAvoid       []string `json:"region_avoid"`
	ResolutionInclude []string `json:"resolution_include"`
	ResolutionAvoid   []string `json:"resolution_avoid"`
	ThreeDInclude     []string `json:"three_d_include"`
	ThreeDAvoid       []string `json:"three_d_avoid"`
	GroupInclude      []string `json:"group_include"`
	GroupAvoid        []string `json:"group_avoid"`
	DubbedAvoid       *bool    `json:"dubbed_avoid,omitempty"`
	HardcodedAvoid    *bool    `json:"hardcoded_avoid,omitempty"`
	ProperInclude     *bool    `json:"proper_include,omitempty"`
	RepackInclude     *bool    `json:"repack_include,omitempty"`
	RepackAvoid       *bool    `json:"repack_avoid,omitempty"`
	ExtendedInclude   *bool    `json:"extended_include,omitempty"`
	UnratedInclude    *bool    `json:"unrated_include,omitempty"`

	MinSizeGB float64 `json:"min_size_gb"`
	MaxSizeGB float64 `json:"max_size_gb"`
	MinYear   int     `json:"min_year"`
	MaxYear   int     `json:"max_year"`

	MinAgeHours float64 `json:"min_age_hours"`
	MaxAgeHours float64 `json:"max_age_hours"`

	KeywordsExcluded []string `json:"keywords_excluded,omitempty"`
	KeywordsRequired []string `json:"keywords_required,omitempty"`

	RegexExcluded []string `json:"regex_excluded,omitempty"`
	RegexRequired []string `json:"regex_required,omitempty"`

	AvailNZBIncluded       []string `json:"availnzb_included,omitempty"`
	AvailNZBRequired       []string `json:"availnzb_required,omitempty"`
	AvailNZBExcluded       []string `json:"availnzb_excluded,omitempty"`
	AvailNZBRequiredLegacy *bool    `json:"availnzb_required_legacy,omitempty"`

	SizePerResolution map[string]SizeRange `json:"size_per_resolution,omitempty"`

	MinBitrateKbps float64 `json:"min_bitrate_kbps"`
	MaxBitrateKbps float64 `json:"max_bitrate_kbps"`

	AllowedQualities  []string `json:"allowed_qualities"`
	BlockedQualities  []string `json:"blocked_qualities"`
	MinResolution     string   `json:"min_resolution"`
	MaxResolution     string   `json:"max_resolution"`
	AllowedCodecs     []string `json:"allowed_codecs"`
	BlockedCodecs     []string `json:"blocked_codecs"`
	RequiredAudio     []string `json:"required_audio"`
	AllowedAudio      []string `json:"allowed_audio"`
	MinChannels       string   `json:"min_channels"`
	RequireHDR        bool     `json:"require_hdr"`
	AllowedHDR        []string `json:"allowed_hdr"`
	BlockedHDR        []string `json:"blocked_hdr"`
	BlockSDR          bool     `json:"block_sdr"`
	RequiredLanguages []string `json:"required_languages"`
	AllowedLanguages  []string `json:"allowed_languages"`
	BlockDubbed       bool     `json:"block_dubbed"`
	BlockCam          bool     `json:"block_cam"`
	RequireProper     bool     `json:"require_proper"`
	AllowRepack       bool     `json:"allow_repack"`
	BlockHardcoded    bool     `json:"block_hardcoded"`
	MinBitDepth       string   `json:"min_bit_depth"`
	BlockedGroups     []string `json:"blocked_groups"`
}

func (c *FilterConfig) UnmarshalJSON(data []byte) error {
	var raw filterConfigRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	c.AudioIncluded = raw.AudioIncluded
	c.BitDepthIncluded = raw.BitDepthIncluded
	c.ChannelsIncluded = raw.ChannelsIncluded
	c.CodecIncluded = raw.CodecIncluded
	c.ContainerIncluded = raw.ContainerIncluded
	c.EditionIncluded = raw.EditionIncluded
	c.HDRIncluded = raw.HDRIncluded
	c.LanguagesIncluded = raw.LanguagesIncluded
	c.NetworkIncluded = raw.NetworkIncluded
	c.QualityIncluded = raw.QualityIncluded
	c.RegionIncluded = raw.RegionIncluded
	c.ResolutionIncluded = raw.ResolutionIncluded
	c.ThreeDIncluded = raw.ThreeDIncluded
	c.GroupIncluded = raw.GroupIncluded

	c.AudioRequired = firstNonEmpty(raw.AudioRequired, raw.AudioInclude, raw.RequiredAudio, raw.AllowedAudio)
	c.BitDepthRequired = firstNonEmpty(raw.BitDepthRequired, raw.BitDepthInclude, condSlice(raw.MinBitDepth != "", []string{raw.MinBitDepth}))
	c.ChannelsRequired = firstNonEmpty(raw.ChannelsRequired, raw.ChannelsInclude, condSlice(raw.MinChannels != "", []string{raw.MinChannels}))
	c.CodecRequired = firstNonEmpty(raw.CodecRequired, raw.CodecInclude, raw.AllowedCodecs)
	c.ContainerRequired = firstNonEmpty(raw.ContainerRequired, raw.ContainerInclude)
	c.EditionRequired = firstNonEmpty(raw.EditionRequired, raw.EditionInclude)
	c.HDRRequired = firstNonEmpty(raw.HDRRequired, raw.HDRInclude, raw.AllowedHDR)
	if raw.RequireHDR && len(c.HDRRequired) == 0 {
		c.HDRRequired = []string{"HDR", "DV", "HDR10+", "3D"}
	}
	c.LanguagesRequired = firstNonEmpty(raw.LanguagesRequired, raw.LanguagesInclude, raw.RequiredLanguages, raw.AllowedLanguages)
	c.NetworkRequired = firstNonEmpty(raw.NetworkRequired, raw.NetworkInclude)
	c.QualityRequired = firstNonEmpty(raw.QualityRequired, raw.QualityInclude, raw.AllowedQualities)
	c.RegionRequired = firstNonEmpty(raw.RegionRequired, raw.RegionInclude)
	c.ResolutionRequired = firstNonEmpty(raw.ResolutionRequired, raw.ResolutionInclude, condSlice(raw.MinResolution != "", []string{raw.MinResolution}))
	c.ThreeDRequired = firstNonEmpty(raw.ThreeDRequired, raw.ThreeDInclude)
	c.GroupRequired = firstNonEmpty(raw.GroupRequired, raw.GroupInclude)

	c.AudioExcluded = firstNonEmpty(raw.AudioExcluded, raw.AudioAvoid)
	c.BitDepthExcluded = firstNonEmpty(raw.BitDepthExcluded, raw.BitDepthAvoid)
	c.ChannelsExcluded = firstNonEmpty(raw.ChannelsExcluded, raw.ChannelsAvoid)
	c.CodecExcluded = firstNonEmpty(raw.CodecExcluded, raw.CodecAvoid, raw.BlockedCodecs)
	c.ContainerExcluded = firstNonEmpty(raw.ContainerExcluded, raw.ContainerAvoid)
	c.EditionExcluded = firstNonEmpty(raw.EditionExcluded, raw.EditionAvoid)
	c.HDRExcluded = firstNonEmpty(raw.HDRExcluded, raw.HDRAvoid, raw.BlockedHDR)
	if raw.BlockSDR && len(c.HDRExcluded) == 0 {
		c.HDRExcluded = []string{"SDR"}
	}
	c.LanguagesExcluded = firstNonEmpty(raw.LanguagesExcluded, raw.LanguagesAvoid)
	c.NetworkExcluded = firstNonEmpty(raw.NetworkExcluded, raw.NetworkAvoid)
	c.QualityExcluded = firstNonEmpty(raw.QualityExcluded, raw.QualityAvoid, raw.BlockedQualities)
	if raw.BlockCam && len(c.QualityExcluded) == 0 {
		c.QualityExcluded = []string{"CAM", "TeleSync", "TeleCine", "SCR"}
	}
	c.RegionExcluded = firstNonEmpty(raw.RegionExcluded, raw.RegionAvoid)
	c.ResolutionExcluded = firstNonEmpty(raw.ResolutionExcluded, raw.ResolutionAvoid)
	c.ThreeDExcluded = firstNonEmpty(raw.ThreeDExcluded, raw.ThreeDAvoid)
	c.GroupExcluded = firstNonEmpty(raw.GroupExcluded, raw.GroupAvoid, raw.BlockedGroups)

	c.DubbedExcluded = firstBoolPtr(raw.DubbedExcluded, raw.DubbedAvoid)
	if c.DubbedExcluded == nil && raw.BlockDubbed {
		t := true
		c.DubbedExcluded = &t
	}
	c.HardcodedExcluded = firstBoolPtr(raw.HardcodedExcluded, raw.HardcodedAvoid)
	if c.HardcodedExcluded == nil && raw.BlockHardcoded {
		t := true
		c.HardcodedExcluded = &t
	}
	c.ProperRequired = firstBoolPtr(raw.ProperRequired, raw.ProperInclude)
	if c.ProperRequired == nil && raw.RequireProper {
		t := true
		c.ProperRequired = &t
	}
	c.RepackRequired = firstBoolPtr(raw.RepackRequired, raw.RepackInclude)
	c.RepackExcluded = firstBoolPtr(raw.RepackExcluded, raw.RepackAvoid)
	if c.RepackRequired == nil && c.RepackExcluded == nil && raw.AllowRepack {
		t := true
		c.RepackRequired = &t
	}
	c.ExtendedRequired = firstBoolPtr(raw.ExtendedRequired, raw.ExtendedInclude)
	c.UnratedRequired = firstBoolPtr(raw.UnratedRequired, raw.UnratedInclude)

	c.MinSizeGB = raw.MinSizeGB
	c.MaxSizeGB = raw.MaxSizeGB
	c.MinYear = raw.MinYear
	c.MaxYear = raw.MaxYear
	c.MinAgeHours = raw.MinAgeHours
	c.MaxAgeHours = raw.MaxAgeHours
	c.KeywordsExcluded = raw.KeywordsExcluded
	c.KeywordsRequired = raw.KeywordsRequired
	c.RegexExcluded = raw.RegexExcluded
	c.RegexRequired = raw.RegexRequired
	c.AvailNZBIncluded = raw.AvailNZBIncluded
	c.AvailNZBRequired = raw.AvailNZBRequired
	c.AvailNZBExcluded = raw.AvailNZBExcluded

	if len(c.AvailNZBRequired) == 0 && raw.AvailNZBRequiredLegacy != nil && *raw.AvailNZBRequiredLegacy {
		c.AvailNZBRequired = []string{"available"}
	}
	c.SizePerResolution = raw.SizePerResolution
	c.MinBitrateKbps = raw.MinBitrateKbps
	c.MaxBitrateKbps = raw.MaxBitrateKbps

	c.LanguagesIncluded = pttoptions.NormalizeLanguageSlice(c.LanguagesIncluded)
	c.LanguagesRequired = pttoptions.NormalizeLanguageSlice(c.LanguagesRequired)
	c.LanguagesExcluded = pttoptions.NormalizeLanguageSlice(c.LanguagesExcluded)
	return nil
}

func firstNonEmpty(slices ...[]string) []string {
	for _, s := range slices {
		if len(s) > 0 {
			return s
		}
	}
	return nil
}

func condSlice(ok bool, s []string) []string {
	if !ok {
		return nil
	}
	return s
}

func firstBoolPtr(ptrs ...*bool) *bool {
	for _, p := range ptrs {
		if p != nil {
			return p
		}
	}
	return nil
}

type sortConfigRaw struct {
	PreferredResolution []string `json:"preferred_resolution"`
	PreferredCodec      []string `json:"preferred_codec"`
	PreferredAudio      []string `json:"preferred_audio"`
	PreferredQuality    []string `json:"preferred_quality"`
	PreferredVisualTag  []string `json:"preferred_visual_tag"`
	PreferredChannels   []string `json:"preferred_channels"`
	PreferredBitDepth   []string `json:"preferred_bit_depth"`
	PreferredContainer  []string `json:"preferred_container"`
	PreferredLanguages  []string `json:"preferred_languages"`
	PreferredGroup      []string `json:"preferred_group"`
	PreferredEdition    []string `json:"preferred_edition"`
	PreferredNetwork    []string `json:"preferred_network"`
	PreferredRegion     []string `json:"preferred_region"`
	PreferredThreeD     []string `json:"preferred_three_d"`
	PreferredAvailNZB   []string `json:"preferred_availnzb,omitempty"`

	ResolutionOrder []string `json:"resolution_order"`
	CodecOrder      []string `json:"codec_order"`
	AudioOrder      []string `json:"audio_order"`
	QualityOrder    []string `json:"quality_order"`
	VisualTagOrder  []string `json:"visual_tag_order"`
	ChannelsOrder   []string `json:"channels_order"`
	BitDepthOrder   []string `json:"bit_depth_order"`
	ContainerOrder  []string `json:"container_order"`
	LanguagesOrder  []string `json:"languages_order"`
	GroupOrder      []string `json:"group_order"`
	EditionOrder    []string `json:"edition_order"`
	NetworkOrder    []string `json:"network_order"`
	RegionOrder     []string `json:"region_order"`
	ThreeDOrder     []string `json:"three_d_order"`

	GrabWeight        float64  `json:"grab_weight"`
	AgeWeight         float64  `json:"age_weight"`
	KeywordsPreferred []string `json:"keywords_preferred,omitempty"`
	KeywordsWeight    float64  `json:"keywords_weight,omitempty"`
	RegexPreferred    []string `json:"regex_preferred,omitempty"`
	RegexWeight       float64  `json:"regex_weight,omitempty"`
	AvailNZBWeight    float64  `json:"availnzb_weight,omitempty"`
	SortCriteriaOrder []string `json:"sort_criteria_order,omitempty"`

	UseCustomScoring *bool    `json:"use_custom_scoring,omitempty"`
	ResolutionWeight float64  `json:"resolution_weight,omitempty"`
	CodecWeight      float64  `json:"codec_weight,omitempty"`
	AudioWeight      float64  `json:"audio_weight,omitempty"`
	QualityWeight    float64  `json:"quality_weight,omitempty"`
	VisualTagWeight  float64  `json:"visual_tag_weight,omitempty"`
	ChannelsWeight   float64  `json:"channels_weight,omitempty"`
	BitDepthWeight   float64  `json:"bit_depth_weight,omitempty"`
	ContainerWeight  float64  `json:"container_weight,omitempty"`
	LanguagesWeight  float64  `json:"languages_weight,omitempty"`
	GroupWeight      float64  `json:"group_weight,omitempty"`
	EditionWeight    float64  `json:"edition_weight,omitempty"`
	NetworkWeight    float64  `json:"network_weight,omitempty"`
	RegionWeight     float64  `json:"region_weight,omitempty"`
	ThreeDWeight     float64  `json:"three_d_weight,omitempty"`
	GroupOrderTier1  []string `json:"group_order_tier1,omitempty"`
	GroupOrderTier2  []string `json:"group_order_tier2,omitempty"`
	GroupOrderTier3  []string `json:"group_order_tier3,omitempty"`
	GroupTier1Points int      `json:"group_tier1_points,omitempty"`
	GroupTier2Points int      `json:"group_tier2_points,omitempty"`
	GroupTier3Points int      `json:"group_tier3_points,omitempty"`

	ResolutionWeights        map[string]int `json:"resolution_weights"`
	CodecWeights             map[string]int `json:"codec_weights"`
	AudioWeights             map[string]int `json:"audio_weights"`
	QualityWeights           map[string]int `json:"quality_weights"`
	VisualTagWeights         map[string]int `json:"visual_tag_weights"`
	LegacyPreferredGroups    []string       `json:"preferred_groups"`
	LegacyPreferredLanguages []string       `json:"preferred_languages_legacy"`
}

func (c *SortConfig) UnmarshalJSON(data []byte) error {
	var raw sortConfigRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	c.PreferredResolution = firstNonEmpty(raw.PreferredResolution, raw.ResolutionOrder)
	if len(c.PreferredResolution) == 0 && len(raw.ResolutionWeights) > 0 {
		c.PreferredResolution = mapKeysByValueDesc(raw.ResolutionWeights)
	}
	c.PreferredCodec = firstNonEmpty(raw.PreferredCodec, raw.CodecOrder)
	if len(c.PreferredCodec) == 0 && len(raw.CodecWeights) > 0 {
		c.PreferredCodec = mapKeysByValueDesc(raw.CodecWeights)
	}
	c.PreferredAudio = firstNonEmpty(raw.PreferredAudio, raw.AudioOrder)
	if len(c.PreferredAudio) == 0 && len(raw.AudioWeights) > 0 {
		c.PreferredAudio = mapKeysByValueDesc(raw.AudioWeights)
	}
	c.PreferredQuality = firstNonEmpty(raw.PreferredQuality, raw.QualityOrder)
	if len(c.PreferredQuality) == 0 && len(raw.QualityWeights) > 0 {
		c.PreferredQuality = mapKeysByValueDesc(raw.QualityWeights)
	}
	c.PreferredVisualTag = firstNonEmpty(raw.PreferredVisualTag, raw.VisualTagOrder)
	if len(c.PreferredVisualTag) == 0 && len(raw.VisualTagWeights) > 0 {
		c.PreferredVisualTag = mapKeysByValueDesc(raw.VisualTagWeights)
	}
	c.PreferredChannels = firstNonEmpty(raw.PreferredChannels, raw.ChannelsOrder)
	c.PreferredBitDepth = firstNonEmpty(raw.PreferredBitDepth, raw.BitDepthOrder)
	c.PreferredContainer = firstNonEmpty(raw.PreferredContainer, raw.ContainerOrder)
	c.PreferredLanguages = pttoptions.NormalizeLanguageSlice(firstNonEmpty(raw.PreferredLanguages, raw.LanguagesOrder, raw.LegacyPreferredLanguages))
	c.PreferredGroup = firstNonEmpty(raw.PreferredGroup, raw.GroupOrder, raw.LegacyPreferredGroups)
	c.PreferredEdition = firstNonEmpty(raw.PreferredEdition, raw.EditionOrder)
	c.PreferredNetwork = firstNonEmpty(raw.PreferredNetwork, raw.NetworkOrder)
	c.PreferredRegion = firstNonEmpty(raw.PreferredRegion, raw.RegionOrder)
	c.PreferredThreeD = firstNonEmpty(raw.PreferredThreeD, raw.ThreeDOrder)
	c.PreferredAvailNZB = raw.PreferredAvailNZB

	if raw.GrabWeight != 0 {
		c.GrabWeight = raw.GrabWeight
	} else {
		c.GrabWeight = 0.5
	}
	if raw.AgeWeight != 0 {
		c.AgeWeight = raw.AgeWeight
	} else {
		c.AgeWeight = 1.0
	}
	c.KeywordsPreferred = raw.KeywordsPreferred
	c.KeywordsWeight = raw.KeywordsWeight
	c.RegexPreferred = raw.RegexPreferred
	c.RegexWeight = raw.RegexWeight
	c.AvailNZBWeight = raw.AvailNZBWeight
	c.SortCriteriaOrder = raw.SortCriteriaOrder
	c.UseCustomScoring = raw.UseCustomScoring
	c.ResolutionWeight = raw.ResolutionWeight
	c.CodecWeight = raw.CodecWeight
	c.AudioWeight = raw.AudioWeight
	c.QualityWeight = raw.QualityWeight
	c.VisualTagWeight = raw.VisualTagWeight
	c.ChannelsWeight = raw.ChannelsWeight
	c.BitDepthWeight = raw.BitDepthWeight
	c.ContainerWeight = raw.ContainerWeight
	c.LanguagesWeight = raw.LanguagesWeight
	c.GroupWeight = raw.GroupWeight
	c.EditionWeight = raw.EditionWeight
	c.NetworkWeight = raw.NetworkWeight
	c.RegionWeight = raw.RegionWeight
	c.ThreeDWeight = raw.ThreeDWeight
	c.GroupOrderTier1 = raw.GroupOrderTier1
	c.GroupOrderTier2 = raw.GroupOrderTier2
	c.GroupOrderTier3 = raw.GroupOrderTier3
	if len(raw.GroupOrderTier1) > 0 || len(raw.GroupOrderTier2) > 0 || len(raw.GroupOrderTier3) > 0 {
		c.GroupTier1Points = raw.GroupTier1Points
		if c.GroupTier1Points == 0 {
			c.GroupTier1Points = 30
		}
		c.GroupTier2Points = raw.GroupTier2Points
		if c.GroupTier2Points == 0 {
			c.GroupTier2Points = 15
		}
		c.GroupTier3Points = raw.GroupTier3Points
		if c.GroupTier3Points == 0 {
			c.GroupTier3Points = 5
		}
	}
	return nil
}

func mapKeysByValueDesc(m map[string]int) []string {
	type kv struct {
		k string
		v int
	}
	var s []kv
	for k, v := range m {
		s = append(s, kv{k, v})
	}
	sort.Slice(s, func(i, j int) bool { return s[i].v > s[j].v })
	out := make([]string, len(s))
	for i := range s {
		out[i] = s[i].k
	}
	return out
}
