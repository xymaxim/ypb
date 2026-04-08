package playback

import (
	internalplayback "github.com/xymaxim/ypb/internal/playback"
	"github.com/xymaxim/ypb/internal/playback/info"
	"github.com/xymaxim/ypb/internal/playback/segment"
)

type Playbacker = internalplayback.Playbacker

type SequenceNumber = internalplayback.SequenceNumber

type SegmentMetadataFetchError = internalplayback.SegmentMetadataFetchError

type RewindMoment = internalplayback.RewindMoment

type RewindInterval = internalplayback.RewindInterval

type SegmentMetadata = segment.Metadata

type VideoInformation = info.VideoInformation
