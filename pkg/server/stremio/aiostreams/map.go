package aiostreams

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"streamnzb/pkg/core/config"
	"streamnzb/pkg/core/logger"
)

const maxConfigBytes = 32 * 1024

// DecodeAndMap decodes the base64url-encoded config param and maps it to
// RequestStreamConfig (FilterConfig + SortConfig). Returns nil on decode or
// mapping failure (logs and continues without override).
func DecodeAndMap(configBase64 string) *RequestStreamConfig {
	if configBase64 == "" {
		return nil
	}
	if len(configBase64) > maxConfigBytes {
		logger.Debug("AIOStreams config too large", "len", len(configBase64), "max", maxConfigBytes)
		return nil
	}
	decoded, err := decodeBase64(configBase64)
	if err != nil {
		logger.Debug("AIOStreams config decode failed", "err", err)
		return nil
	}
	var p Payload
	if err := json.Unmarshal(decoded, &p); err != nil {
		logger.Debug("AIOStreams config JSON unmarshal failed", "err", err)
		return nil
	}
	return MapPayload(&p)
}

func decodeBase64(s string) ([]byte, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return base64.StdEncoding.DecodeString(s)
	}
	return b, nil
}

// MapPayload converts an AIOStreams payload into StreamNZB FilterConfig and SortConfig.
func MapPayload(p *Payload) *RequestStreamConfig {
	if p == nil {
		return nil
	}
	filters := mapFilters(p)
	sorting := mapSorting(p)
	// If both are effectively empty, no override
	if filters == nil && sorting == nil {
		return nil
	}
	out := &RequestStreamConfig{}
	if filters != nil {
		out.Filters = filters
	}
	if sorting != nil {
		out.Sorting = sorting
	}
	return out
}

func mapFilters(p *Payload) *config.FilterConfig {
	f := &config.FilterConfig{}
	hasAny := false

	setSlice := func(dst *[]string, src []string) {
		if len(src) == 0 {
			return
		}
		*dst = append([]string(nil), src...)
		hasAny = true
	}

	setSlice(&f.ResolutionAvoid, p.ExcludedResolutions)
	setSlice(&f.ResolutionInclude, p.IncludedResolutions)
	// requiredResolutions: StreamNZB has no "required" for resolution; treat as include
	if len(p.RequiredResolutions) > 0 {
		f.ResolutionInclude = append(f.ResolutionInclude, p.RequiredResolutions...)
		hasAny = true
	}

	setSlice(&f.QualityAvoid, p.ExcludedQualities)
	setSlice(&f.QualityInclude, p.IncludedQualities)
	if len(p.RequiredQualities) > 0 {
		f.QualityInclude = append(f.QualityInclude, p.RequiredQualities...)
		hasAny = true
	}

	setSlice(&f.LanguagesAvoid, p.ExcludedLanguages)
	setSlice(&f.LanguagesInclude, p.IncludedLanguages)
	if len(p.RequiredLanguages) > 0 {
		f.LanguagesInclude = append(f.LanguagesInclude, p.RequiredLanguages...)
		hasAny = true
	}

	// Visual tags: AIOStreams "3D" -> ThreeDAvoid; other (HDR10, DV, etc.) -> HDRAvoid
	for _, t := range p.ExcludedVisualTags {
		key := strings.TrimSpace(strings.ToLower(t))
		if key == "3d" {
			f.ThreeDAvoid = append(f.ThreeDAvoid, strings.TrimSpace(t))
		} else {
			f.HDRAvoid = append(f.HDRAvoid, strings.TrimSpace(t))
		}
		hasAny = true
	}
	setSlice(&f.HDRInclude, p.IncludedVisualTags)
	setSlice(&f.ThreeDInclude, p.IncludedVisualTags)

	setSlice(&f.AudioAvoid, p.ExcludedAudioTags)
	setSlice(&f.AudioInclude, p.IncludedAudioTags)
	setSlice(&f.ChannelsAvoid, p.ExcludedAudioChannels)
	setSlice(&f.ChannelsInclude, p.IncludedAudioChannels)

	setSlice(&f.CodecAvoid, p.ExcludedEncodes)
	setSlice(&f.CodecInclude, p.IncludedEncodes)
	setSlice(&f.ContainerAvoid, p.ExcludedStreamTypes)
	setSlice(&f.ContainerInclude, p.IncludedStreamTypes)

	setSlice(&f.GroupAvoid, p.ExcludedReleaseGroups)
	setSlice(&f.GroupInclude, p.IncludedReleaseGroups)
	if len(p.RequiredReleaseGroups) > 0 {
		f.GroupInclude = append(f.GroupInclude, p.RequiredReleaseGroups...)
		hasAny = true
	}

	if !hasAny {
		return nil
	}
	return f
}

func mapSorting(p *Payload) *config.SortConfig {
	s := &config.SortConfig{}
	// Default weights so order lists matter
	s.GrabWeight = 1.0
	s.AgeWeight = 0.01
	hasAny := false

	setOrder := func(dst *[]string, src []string) {
		if len(src) == 0 {
			return
		}
		*dst = append([]string(nil), src...)
		hasAny = true
	}

	setOrder(&s.ResolutionOrder, p.PreferredResolutions)
	setOrder(&s.QualityOrder, p.PreferredQualities)
	setOrder(&s.VisualTagOrder, p.PreferredVisualTags)
	setOrder(&s.AudioOrder, p.PreferredAudioTags)
	setOrder(&s.ChannelsOrder, p.PreferredAudioChannels)
	setOrder(&s.ContainerOrder, p.PreferredStreamTypes)
	setOrder(&s.CodecOrder, p.PreferredEncodes)
	setOrder(&s.LanguagesOrder, p.PreferredLanguages)
	setOrder(&s.GroupOrder, p.PreferredReleaseGroups)

	if !hasAny {
		return nil
	}
	return s
}
