package service

// LeaderboardMeta 9 个榜单的元信息。
type LeaderboardMeta struct {
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	Emoji       string   `json:"emoji"`
	Description string   `json:"description"`
	Ranges      []string `json:"ranges"`
}

// AllMeta 9 个榜单的元信息全集。
var AllMeta = []LeaderboardMeta{
	{Type: "rich", Name: "土豪榜", Emoji: "💰", Description: "钱包余额最多 · 谁是钱袋子之王", Ranges: []string{"all"}},
	{Type: "spender", Name: "散财榜", Emoji: "💸", Description: "累计消费最多 · 挥金如土", Ranges: []string{"today", "7d", "30d", "all"}},
	{Type: "topup", Name: "充值榜", Emoji: "💎", Description: "充值金额最多", Ranges: []string{"7d", "30d", "all"}},
	{Type: "foodie", Name: "吃货榜", Emoji: "🍴", Description: "调用次数最多 · 吃货之王", Ranges: []string{"today", "7d", "30d"}},
	{Type: "loyal", Name: "死忠粉榜", Emoji: "🎯", Description: "对单一模型最专一", Ranges: []string{"7d", "30d", "all"}},
	{Type: "gourmet", Name: "美食家榜", Emoji: "🌈", Description: "尝过的模型种类最多", Ranges: []string{"7d", "30d", "all"}},
	{Type: "biteking", Name: "单笔王", Emoji: "🔥", Description: "单次最大消耗 · 豪赌一把", Ranges: []string{"7d", "30d", "all"}},
	{Type: "tokens", Name: "吞噬榜", Emoji: "⚡", Description: "累计 token 消耗最多", Ranges: []string{"today", "7d", "30d"}},
	{Type: "nightowl", Name: "熬夜冠军", Emoji: "🌙", Description: "凌晨 0-5 点调用最多", Ranges: []string{"7d", "30d"}},
}

func MetaByType(t string) (LeaderboardMeta, bool) {
	for _, m := range AllMeta {
		if m.Type == t {
			return m, true
		}
	}
	return LeaderboardMeta{}, false
}

func ValidType(t string) bool { _, ok := MetaByType(t); return ok }

func RangeSupported(t, r string) bool {
	m, ok := MetaByType(t)
	if !ok {
		return false
	}
	for _, x := range m.Ranges {
		if x == r {
			return true
		}
	}
	return false
}
