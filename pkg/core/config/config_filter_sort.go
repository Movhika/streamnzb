package config

import (
	"encoding/json"
	"sort"
)

// FilterConfig holds Include/Avoid per PTT category (typed options). Replaces legacy allowed/blocked/required.
type FilterConfig struct {
	AudioInclude     []string `json:"audio_include"`
	AudioAvoid       []string `json:"audio_avoid"`
	BitDepthInclude  []string `json:"bit_depth_include"`
	BitDepthAvoid    []string `json:"bit_depth_avoid"`
	ChannelsInclude  []string `json:"channels_include"`
	ChannelsAvoid    []string `json:"channels_avoid"`
	CodecInclude     []string `json:"codec_include"`
	CodecAvoid       []string `json:"codec_avoid"`
	ContainerInclude []string `json:"container_include"`
	ContainerAvoid   []string `json:"container_avoid"`
	EditionInclude   []string `json:"edition_include"`
	EditionAvoid     []string `json:"edition_avoid"`
	HDRInclude       []string `json:"hdr_include"`
	HDRAvoid         []string `json:"hdr_avoid"`
	LanguagesInclude []string `json:"languages_include"`
	LanguagesAvoid   []string `json:"languages_avoid"`
	NetworkInclude   []string `json:"network_include"`
	NetworkAvoid     []string `json:"network_avoid"`
	QualityInclude   []string `json:"quality_include"`
	QualityAvoid     []string `json:"quality_avoid"`
	RegionInclude    []string `json:"region_include"`
	RegionAvoid      []string `json:"region_avoid"`
	ResolutionInclude []string `json:"resolution_include"`
	ResolutionAvoid   []string `json:"resolution_avoid"`
	ThreeDInclude    []string `json:"three_d_include"`
	ThreeDAvoid      []string `json:"three_d_avoid"`
	GroupInclude     []string `json:"group_include"`
	GroupAvoid       []string `json:"group_avoid"`

	DubbedAvoid     *bool `json:"dubbed_avoid,omitempty"`
	HardcodedAvoid  *bool `json:"hardcoded_avoid,omitempty"`
	ProperInclude   *bool `json:"proper_include,omitempty"`
	RepackInclude   *bool `json:"repack_include,omitempty"`
	RepackAvoid     *bool `json:"repack_avoid,omitempty"`
	ExtendedInclude *bool `json:"extended_include,omitempty"`
	UnratedInclude  *bool `json:"unrated_include,omitempty"`

	MinSizeGB float64 `json:"min_size_gb"`
	MaxSizeGB float64 `json:"max_size_gb"`
	MinYear   int     `json:"min_year"`
	MaxYear   int     `json:"max_year"`
}

// SortConfig holds ordered lists per category; points scale 10 (first) to 0 (last). Replaces legacy weight maps.
// When UseCustomScoring is true, each category's 0–10 contribution is multiplied by its *_weight (default 1.0),
// and release group can use three tiers with separate points instead of one ordered list.
type SortConfig struct {
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

	GrabWeight float64 `json:"grab_weight"`
	AgeWeight  float64 `json:"age_weight"`

	// Custom scoring: when true, category weights multiply each category's 0–10 points; group can use tiers.
	UseCustomScoring *bool `json:"use_custom_scoring,omitempty"`
	// Category weights (default 1.0 when use_custom_scoring). Only used when UseCustomScoring is true.
	ResolutionWeight  float64 `json:"resolution_weight,omitempty"`
	CodecWeight       float64 `json:"codec_weight,omitempty"`
	AudioWeight       float64 `json:"audio_weight,omitempty"`
	QualityWeight     float64 `json:"quality_weight,omitempty"`
	VisualTagWeight   float64 `json:"visual_tag_weight,omitempty"`
	ChannelsWeight    float64 `json:"channels_weight,omitempty"`
	BitDepthWeight    float64 `json:"bit_depth_weight,omitempty"`
	ContainerWeight   float64 `json:"container_weight,omitempty"`
	LanguagesWeight   float64 `json:"languages_weight,omitempty"`
	GroupWeight       float64 `json:"group_weight,omitempty"`
	EditionWeight     float64 `json:"edition_weight,omitempty"`
	NetworkWeight     float64 `json:"network_weight,omitempty"`
	RegionWeight      float64 `json:"region_weight,omitempty"`
	ThreeDWeight      float64 `json:"three_d_weight,omitempty"`
	// Release group tiers: when any tier list is non-empty, group score = tier points (whole-word match). Else use GroupOrder.
	GroupOrderTier1   []string `json:"group_order_tier1,omitempty"`
	GroupOrderTier2   []string `json:"group_order_tier2,omitempty"`
	GroupOrderTier3   []string `json:"group_order_tier3,omitempty"`
	GroupTier1Points   int      `json:"group_tier1_points,omitempty"`
	GroupTier2Points   int      `json:"group_tier2_points,omitempty"`
	GroupTier3Points   int      `json:"group_tier3_points,omitempty"`
}

// filterConfigRaw is used to unmarshal both legacy and new JSON into FilterConfig.
type filterConfigRaw struct {
	// New
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
	GroupInclude     []string `json:"group_include"`
	GroupAvoid       []string `json:"group_avoid"`
	DubbedAvoid      *bool    `json:"dubbed_avoid,omitempty"`
	HardcodedAvoid   *bool    `json:"hardcoded_avoid,omitempty"`
	ProperInclude    *bool    `json:"proper_include,omitempty"`
	RepackInclude    *bool   `json:"repack_include,omitempty"`
	RepackAvoid      *bool   `json:"repack_avoid,omitempty"`
	ExtendedInclude  *bool   `json:"extended_include,omitempty"`
	UnratedInclude    *bool   `json:"unrated_include,omitempty"`
	MinSizeGB         float64 `json:"min_size_gb"`
	MaxSizeGB         float64 `json:"max_size_gb"`
	MinYear           int     `json:"min_year"`
	MaxYear           int     `json:"max_year"`
	// Legacy
	AllowedQualities   []string `json:"allowed_qualities"`
	BlockedQualities   []string `json:"blocked_qualities"`
	MinResolution      string   `json:"min_resolution"`
	MaxResolution      string   `json:"max_resolution"`
	AllowedCodecs      []string `json:"allowed_codecs"`
	BlockedCodecs      []string `json:"blocked_codecs"`
	RequiredAudio      []string `json:"required_audio"`
	AllowedAudio       []string `json:"allowed_audio"`
	MinChannels        string   `json:"min_channels"`
	RequireHDR         bool     `json:"require_hdr"`
	AllowedHDR         []string `json:"allowed_hdr"`
	BlockedHDR         []string `json:"blocked_hdr"`
	BlockSDR           bool     `json:"block_sdr"`
	RequiredLanguages  []string `json:"required_languages"`
	AllowedLanguages   []string `json:"allowed_languages"`
	BlockDubbed        bool     `json:"block_dubbed"`
	BlockCam           bool     `json:"block_cam"`
	RequireProper      bool     `json:"require_proper"`
	AllowRepack        bool     `json:"allow_repack"`
	BlockHardcoded     bool     `json:"block_hardcoded"`
	MinBitDepth        string   `json:"min_bit_depth"`
	BlockedGroups      []string `json:"blocked_groups"`
}

// UnmarshalJSON supports both new (include/avoid) and legacy (allowed/blocked) filter JSON.
func (c *FilterConfig) UnmarshalJSON(data []byte) error {
	var raw filterConfigRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	// New fields take precedence; fall back to legacy
	c.AudioInclude = firstNonEmpty(raw.AudioInclude, raw.RequiredAudio, raw.AllowedAudio)
	c.AudioAvoid = firstNonEmpty(raw.AudioAvoid, nil)
	c.BitDepthInclude = firstNonEmpty(raw.BitDepthInclude, condSlice(raw.MinBitDepth != "", []string{raw.MinBitDepth}))
	c.BitDepthAvoid = raw.BitDepthAvoid
	c.ChannelsInclude = firstNonEmpty(raw.ChannelsInclude, condSlice(raw.MinChannels != "", []string{raw.MinChannels}))
	c.ChannelsAvoid = raw.ChannelsAvoid
	c.CodecInclude = firstNonEmpty(raw.CodecInclude, raw.AllowedCodecs)
	c.CodecAvoid = firstNonEmpty(raw.CodecAvoid, raw.BlockedCodecs)
	c.ContainerInclude = raw.ContainerInclude
	c.ContainerAvoid = raw.ContainerAvoid
	c.EditionInclude = raw.EditionInclude
	c.EditionAvoid = raw.EditionAvoid
	c.HDRInclude = firstNonEmpty(raw.HDRInclude, raw.AllowedHDR)
	if raw.RequireHDR && len(c.HDRInclude) == 0 {
		c.HDRInclude = []string{"HDR", "DV", "HDR10+", "3D"}
	}
	c.HDRAvoid = firstNonEmpty(raw.HDRAvoid, raw.BlockedHDR)
	if raw.BlockSDR && len(c.HDRAvoid) == 0 {
		c.HDRAvoid = []string{"SDR"}
	}
	c.LanguagesInclude = firstNonEmpty(raw.LanguagesInclude, raw.RequiredLanguages, raw.AllowedLanguages)
	c.LanguagesAvoid = raw.LanguagesAvoid
	c.NetworkInclude = raw.NetworkInclude
	c.NetworkAvoid = raw.NetworkAvoid
	c.QualityInclude = firstNonEmpty(raw.QualityInclude, raw.AllowedQualities)
	c.QualityAvoid = firstNonEmpty(raw.QualityAvoid, raw.BlockedQualities)
	if raw.BlockCam && len(c.QualityAvoid) == 0 {
		c.QualityAvoid = []string{"CAM", "TeleSync", "TeleCine", "SCR"}
	}
	c.RegionInclude = raw.RegionInclude
	c.RegionAvoid = raw.RegionAvoid
	c.ResolutionInclude = firstNonEmpty(raw.ResolutionInclude, condSlice(raw.MinResolution != "", []string{raw.MinResolution}))
	c.ResolutionAvoid = raw.ResolutionAvoid
	c.ThreeDInclude = raw.ThreeDInclude
	c.ThreeDAvoid = raw.ThreeDAvoid
	c.GroupInclude = raw.GroupInclude
	c.GroupAvoid = firstNonEmpty(raw.GroupAvoid, raw.BlockedGroups)
	if raw.DubbedAvoid != nil {
		c.DubbedAvoid = raw.DubbedAvoid
	} else if raw.BlockDubbed {
		t := true
		c.DubbedAvoid = &t
	}
	if raw.HardcodedAvoid != nil {
		c.HardcodedAvoid = raw.HardcodedAvoid
	} else if raw.BlockHardcoded {
		t := true
		c.HardcodedAvoid = &t
	}
	if raw.ProperInclude != nil {
		c.ProperInclude = raw.ProperInclude
	} else if raw.RequireProper {
		t := true
		c.ProperInclude = &t
	}
	if raw.RepackInclude != nil {
		c.RepackInclude = raw.RepackInclude
	} else if raw.RepackAvoid != nil {
		c.RepackAvoid = raw.RepackAvoid
	} else if raw.AllowRepack {
		t := true
		c.RepackInclude = &t
	}
	c.ExtendedInclude = raw.ExtendedInclude
	c.UnratedInclude = raw.UnratedInclude
	c.MinSizeGB = raw.MinSizeGB
	c.MaxSizeGB = raw.MaxSizeGB
	c.MinYear = raw.MinYear
	c.MaxYear = raw.MaxYear
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

// sortConfigRaw for unmarshaling legacy (weight maps) and new (order slices) sort JSON.
type sortConfigRaw struct {
	ResolutionOrder []string       `json:"resolution_order"`
	CodecOrder      []string       `json:"codec_order"`
	AudioOrder      []string       `json:"audio_order"`
	QualityOrder    []string       `json:"quality_order"`
	VisualTagOrder  []string       `json:"visual_tag_order"`
	ChannelsOrder   []string       `json:"channels_order"`
	BitDepthOrder   []string       `json:"bit_depth_order"`
	ContainerOrder  []string       `json:"container_order"`
	LanguagesOrder  []string       `json:"languages_order"`
	GroupOrder      []string       `json:"group_order"`
	EditionOrder    []string       `json:"edition_order"`
	NetworkOrder    []string       `json:"network_order"`
	RegionOrder     []string       `json:"region_order"`
	ThreeDOrder     []string       `json:"three_d_order"`
	GrabWeight      float64        `json:"grab_weight"`
	AgeWeight       float64        `json:"age_weight"`
	// Custom scoring
	UseCustomScoring *bool    `json:"use_custom_scoring,omitempty"`
	ResolutionWeight  float64 `json:"resolution_weight,omitempty"`
	CodecWeight       float64 `json:"codec_weight,omitempty"`
	AudioWeight       float64 `json:"audio_weight,omitempty"`
	QualityWeight     float64 `json:"quality_weight,omitempty"`
	VisualTagWeight   float64 `json:"visual_tag_weight,omitempty"`
	ChannelsWeight    float64 `json:"channels_weight,omitempty"`
	BitDepthWeight    float64 `json:"bit_depth_weight,omitempty"`
	ContainerWeight   float64 `json:"container_weight,omitempty"`
	LanguagesWeight   float64 `json:"languages_weight,omitempty"`
	GroupWeight       float64 `json:"group_weight,omitempty"`
	EditionWeight     float64 `json:"edition_weight,omitempty"`
	NetworkWeight     float64 `json:"network_weight,omitempty"`
	RegionWeight      float64 `json:"region_weight,omitempty"`
	ThreeDWeight      float64 `json:"three_d_weight,omitempty"`
	GroupOrderTier1   []string `json:"group_order_tier1,omitempty"`
	GroupOrderTier2   []string `json:"group_order_tier2,omitempty"`
	GroupOrderTier3   []string `json:"group_order_tier3,omitempty"`
	GroupTier1Points   int      `json:"group_tier1_points,omitempty"`
	GroupTier2Points   int      `json:"group_tier2_points,omitempty"`
	GroupTier3Points   int      `json:"group_tier3_points,omitempty"`
	// Legacy
	ResolutionWeights   map[string]int `json:"resolution_weights"`
	CodecWeights       map[string]int `json:"codec_weights"`
	AudioWeights       map[string]int `json:"audio_weights"`
	QualityWeights     map[string]int `json:"quality_weights"`
	VisualTagWeights   map[string]int `json:"visual_tag_weights"`
	PreferredGroups    []string       `json:"preferred_groups"`
	PreferredLanguages []string       `json:"preferred_languages"`
}

// UnmarshalJSON supports both new (order slices) and legacy (weight maps) sort JSON.
func (c *SortConfig) UnmarshalJSON(data []byte) error {
	var raw sortConfigRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.ResolutionOrder = raw.ResolutionOrder
	if len(c.ResolutionOrder) == 0 && len(raw.ResolutionWeights) > 0 {
		c.ResolutionOrder = mapKeysByValueDesc(raw.ResolutionWeights)
	}
	c.CodecOrder = raw.CodecOrder
	if len(c.CodecOrder) == 0 && len(raw.CodecWeights) > 0 {
		c.CodecOrder = mapKeysByValueDesc(raw.CodecWeights)
	}
	c.AudioOrder = raw.AudioOrder
	if len(c.AudioOrder) == 0 && len(raw.AudioWeights) > 0 {
		c.AudioOrder = mapKeysByValueDesc(raw.AudioWeights)
	}
	c.QualityOrder = raw.QualityOrder
	if len(c.QualityOrder) == 0 && len(raw.QualityWeights) > 0 {
		c.QualityOrder = mapKeysByValueDesc(raw.QualityWeights)
	}
	c.VisualTagOrder = raw.VisualTagOrder
	if len(c.VisualTagOrder) == 0 && len(raw.VisualTagWeights) > 0 {
		c.VisualTagOrder = mapKeysByValueDesc(raw.VisualTagWeights)
	}
	c.ChannelsOrder = raw.ChannelsOrder
	c.BitDepthOrder = raw.BitDepthOrder
	c.ContainerOrder = raw.ContainerOrder
	c.LanguagesOrder = raw.LanguagesOrder
	if len(c.LanguagesOrder) == 0 {
		c.LanguagesOrder = raw.PreferredLanguages
	}
	c.GroupOrder = raw.GroupOrder
	if len(c.GroupOrder) == 0 {
		c.GroupOrder = raw.PreferredGroups
	}
	c.EditionOrder = raw.EditionOrder
	c.NetworkOrder = raw.NetworkOrder
	c.RegionOrder = raw.RegionOrder
	c.ThreeDOrder = raw.ThreeDOrder
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

// mapKeysByValueDesc returns keys sorted by value descending (highest first).
func mapKeysByValueDesc(m map[string]int) []string {
	type kv struct{ k string; v int }
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
