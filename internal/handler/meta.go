package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/0401lucky/newapi-paihang/internal/config"
	"github.com/0401lucky/newapi-paihang/internal/service"
)

type EmbedDefault struct {
	Tabs     []string `json:"tabs"`
	SiteURL  string   `json:"site_url"`
	SiteName string   `json:"site_name"`
}

type MetaResponse struct {
	Leaderboards []service.LeaderboardMeta `json:"leaderboards"`
	Embed        EmbedDefault              `json:"embed"`
	Version      string                    `json:"version"`
}

func Meta(cfg *config.Config, version string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": MetaResponse{
				Leaderboards: service.AllMeta,
				Embed: EmbedDefault{
					Tabs:     cfg.EmbedTabsDefault,
					SiteURL:  cfg.SiteURL,
					SiteName: cfg.SiteName,
				},
				Version: version,
			},
		})
	}
}
