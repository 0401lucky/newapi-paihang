package repo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuildHiddenClause(t *testing.T) {
	clause, args := BuildHiddenClause(nil)
	assert.Empty(t, clause)
	assert.Empty(t, args)

	clause, args = BuildHiddenClause([]int64{1, 2, 3})
	assert.Equal(t, " AND u.id NOT IN (?,?,?)", clause)
	assert.Equal(t, []any{int64(1), int64(2), int64(3)}, args)
}

func TestRangeStart_Today(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 5, 24, 15, 30, 0, 0, loc)
	ts := rangeStartAt(RangeToday, now)
	expected := time.Date(2026, 5, 24, 0, 0, 0, 0, loc).Unix()
	assert.Equal(t, expected, ts)
}

func TestRangeStart_7d(t *testing.T) {
	now := time.Date(2026, 5, 24, 15, 30, 0, 0, time.UTC)
	ts := rangeStartAt(Range7d, now)
	assert.Equal(t, now.Add(-7*24*time.Hour).Unix(), ts)
}

func TestRangeStart_All(t *testing.T) {
	now := time.Now()
	ts := rangeStartAt(RangeAll, now)
	assert.EqualValues(t, 0, ts)
}

func TestValidRange(t *testing.T) {
	assert.True(t, ValidRange("today"))
	assert.True(t, ValidRange("7d"))
	assert.True(t, ValidRange("30d"))
	assert.True(t, ValidRange("all"))
	assert.False(t, ValidRange("yesterday"))
	assert.False(t, ValidRange(""))
}

func TestFormatUSD(t *testing.T) {
	assert.Equal(t, "$0.00", FormatUSD(0))
	assert.Equal(t, "$2.00", FormatUSD(1000000))
	assert.Equal(t, "$8,432.12", FormatUSD(4216060000))
}

func TestFormatCount(t *testing.T) {
	assert.Equal(t, "0", FormatCount(0))
	assert.Equal(t, "999", FormatCount(999))
	assert.Equal(t, "1,000", FormatCount(1000))
	assert.Equal(t, "12,345,678", FormatCount(12345678))
}
