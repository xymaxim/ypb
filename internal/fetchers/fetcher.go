package fetchers

import (
	"github.com/xymaxim/ypb/internal/info"
)

type Additionals any

type Fetcher interface {
	FetchInfo() (*info.VideoInformation, Additionals, error)
	FetchBaseURLs() (map[string]string, error)
}
