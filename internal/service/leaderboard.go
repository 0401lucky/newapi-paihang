package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/yourname/newapi-leaderboard/internal/cache"
	"github.com/yourname/newapi-leaderboard/internal/config"
	"github.com/yourname/newapi-leaderboard/internal/repo"
)

// Service 业务编排：缓存命中 → repo → 写缓存。
type Service struct {
	db    *sql.DB
	cache *cache.Cache
	cfg   *config.Config
}

func New(db *sql.DB, c *cache.Cache, cfg *config.Config) *Service {
	return &Service{db: db, cache: c, cfg: cfg}
}

// Response 排行榜 API 响应数据部分。
type Response struct {
	Type      string                 `json:"type"`
	Range     string                 `json:"range"`
	Total     int                    `json:"total"`
	Page      int                    `json:"page"`
	PageSize  int                    `json:"page_size"`
	List      []repo.LeaderboardItem `json:"list"`
	UpdatedAt int64                  `json:"updated_at"`
	Cached    bool                   `json:"cached"`
	Stale     bool                   `json:"stale,omitempty"`
}

type fetchResult struct {
	Items []repo.LeaderboardItem
	Total int
	At    int64
}

// GetHiddenUserIDs 合并 env 配置的隐藏列表和动态 extra（admin store）。
func (s *Service) GetHiddenUserIDs(extra []int64) []int64 {
	out := make([]int64, 0, len(s.cfg.HiddenUserIDs)+len(extra))
	out = append(out, s.cfg.HiddenUserIDs...)
	out = append(out, extra...)
	return out
}

// Get 榜单查询统一入口。
func (s *Service) Get(ctx context.Context, lbType, rangeStr string, page, pageSize int, extraHidden []int64) (*Response, error) {
	if !ValidType(lbType) {
		return nil, fmt.Errorf("unknown type: %s", lbType)
	}
	if !RangeSupported(lbType, rangeStr) {
		return nil, fmt.Errorf("range %s not supported for %s", rangeStr, lbType)
	}

	cacheKey := fmt.Sprintf("lb:%s:%s:p%d:s%d", lbType, rangeStr, page, pageSize)
	ttl := time.Duration(s.cfg.CacheTTLLogs) * time.Second
	// rich 总是用 users.quota；spender 仅 all 模式用 users.used_quota
	if lbType == "rich" || (lbType == "spender" && rangeStr == "all") {
		ttl = time.Duration(s.cfg.CacheTTLUsers) * time.Second
	}

	val, stale, err := s.cache.GetOrLoad(cacheKey, ttl, func() (any, error) {
		items, total, err := s.fetch(ctx, lbType, rangeStr, page, pageSize, extraHidden)
		if err != nil {
			return nil, err
		}
		return fetchResult{Items: items, Total: total, At: time.Now().Unix()}, nil
	})
	if err != nil {
		return nil, err
	}
	fr := val.(fetchResult)
	return &Response{
		Type: lbType, Range: rangeStr, Total: fr.Total,
		Page: page, PageSize: pageSize, List: fr.Items,
		UpdatedAt: fr.At, Cached: true, Stale: stale,
	}, nil
}

func (s *Service) fetch(ctx context.Context, t, r string, page, ps int, extraHidden []int64) ([]repo.LeaderboardItem, int, error) {
	q := repo.QueryParams{
		Range:          repo.Range(r),
		Page:           page,
		PageSize:       ps,
		HiddenUserIDs:  s.GetHiddenUserIDs(extraHidden),
		LoyalThreshold: s.cfg.LoyalThreshold,
		LoyalMinCalls:  s.cfg.LoyalMinCalls,
	}
	switch t {
	case "rich":
		return repo.RichTop(ctx, s.db, q)
	case "spender":
		return repo.SpenderTop(ctx, s.db, q)
	case "topup":
		return repo.TopupTop(ctx, s.db, q)
	case "foodie":
		return repo.FoodieTop(ctx, s.db, q)
	case "loyal":
		return repo.LoyalTop(ctx, s.db, q)
	case "gourmet":
		return repo.GourmetTop(ctx, s.db, q)
	case "biteking":
		return repo.BitekingTop(ctx, s.db, q)
	case "tokens":
		return repo.TokensTop(ctx, s.db, q)
	case "nightowl":
		return repo.NightowlTop(ctx, s.db, q)
	}
	return nil, 0, fmt.Errorf("unreachable: %s", t)
}

// PersonalRank 个人查询结果。
type PersonalRank struct {
	UserID int64                    `json:"user_id"`
	Name   string                   `json:"name"`
	Ranks  map[string]PersonalEntry `json:"ranks"`
}

// PersonalEntry 单榜的个人位次。
type PersonalEntry struct {
	Rank  int     `json:"rank"`
	Value float64 `json:"value,omitempty"`
}

// SearchAndRank LIKE 搜索 keyword 候选 + 对每个候选算 9 榜位次。
func (s *Service) SearchAndRank(ctx context.Context, keyword string, extraHidden []int64) ([]PersonalRank, error) {
	hidden := s.GetHiddenUserIDs(extraHidden)
	users, err := repo.SearchUsers(ctx, s.db, keyword, hidden, 10)
	if err != nil {
		return nil, err
	}
	out := make([]PersonalRank, 0, len(users))
	for _, u := range users {
		pr := PersonalRank{UserID: u.ID, Name: u.Name, Ranks: map[string]PersonalEntry{}}
		if rk, _ := repo.RichRankOf(ctx, s.db, u.ID, hidden); rk > 0 {
			pr.Ranks["rich"] = PersonalEntry{Rank: rk}
		}
		if rk, _ := repo.SpenderRankOf(ctx, s.db, u.ID, hidden); rk > 0 {
			pr.Ranks["spender"] = PersonalEntry{Rank: rk}
		}
		for _, m := range []struct{ t, metric, r string }{
			{"foodie", "count", "7d"},
			{"gourmet", "distinct_model", "30d"},
			{"biteking", "max_quota", "30d"},
			{"tokens", "sum_tokens", "7d"},
			{"nightowl", "night_count", "30d"},
		} {
			if rk, _ := repo.LogAggRankOf(ctx, s.db, u.ID, repo.Range(m.r), m.metric, hidden); rk > 0 {
				pr.Ranks[m.t] = PersonalEntry{Rank: rk}
			}
		}
		out = append(out, pr)
	}
	return out, nil
}
