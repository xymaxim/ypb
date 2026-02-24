package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/xymaxim/ypb/internal/playback/info"
)

type jsonInfo struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	ChannelID       string    `json:"channelId"`
	ChannelTitle    string    `json:"channelTitle"`
	ActualStartTime time.Time `json:"actualStartTime"`
}

type InfoHandler struct {
	Info info.VideoInformation
}

func (h *InfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	content := jsonInfo{
		ID:              h.Info.ID,
		Title:           h.Info.Title,
		ChannelID:       h.Info.ChannelID,
		ChannelTitle:    h.Info.ChannelTitle,
		ActualStartTime: h.Info.ActualStartTime,
	}

	err := json.NewEncoder(w).Encode(content)
	if err != nil {
		return fmt.Errorf("writing json response: %w", err)
	}

	return nil
}
