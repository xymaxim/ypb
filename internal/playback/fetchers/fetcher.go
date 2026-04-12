package fetchers

import (
	"context"

	"github.com/xymaxim/ypb/internal/playback/info"
)

type Additionals any

type Fetcher interface {
	FetchInfo(ctx context.Context) (*info.VideoInformation, Additionals, error)
	FetchBaseURLs(ctx context.Context) (map[string]string, error)
}
