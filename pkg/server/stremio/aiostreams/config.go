package aiostreams

import (
	"streamnzb/pkg/core/config"
)

type RequestStreamConfig struct {
	Filters *config.FilterConfig
	Sorting *config.SortConfig
}
