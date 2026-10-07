package models

import (
	"context"
	"fmt"
	"strings"

	"github.com/IM_System/pkg/readquery"
)

type UserReferenceFilter struct {
	UserID, Nickname, Phone string
	Limit                   int32
}
type UserReferenceRow struct {
	ID          string `db:"id"`
	DisplayName string `db:"nickname"`
	Status      int32  `db:"status"`
}

func referenceQuery(table string, f UserReferenceFilter) (string, []any, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		return "", nil, fmt.Errorf("invalid user reference limit")
	}
	selectors := 0
	for _, v := range []string{f.UserID, f.Nickname, f.Phone} {
		if v != "" {
			selectors++
		}
	}
	if selectors != 1 {
		return "", nil, fmt.Errorf("exactly one user reference selector required")
	}
	var predicate string
	var value string
	switch {
	case f.UserID != "":
		predicate = "`id` = ?"
		value = f.UserID
	case f.Phone != "":
		predicate = "`phone` = ?"
		value = f.Phone
	default:
		predicate = "`nickname` like ? escape '='"
		value = "%" + strings.NewReplacer("=", "==", "%", "=%", "_", "=_").Replace(f.Nickname) + "%"
	}
	// Fetch only the minimal projection, not password/phone/avatar. LIMIT is
	// applied by MySQL, with one extra row to prove truncation.
	query := fmt.Sprintf("select `id`, `nickname`, coalesce(`status`, 0) as `status` from %s where %s order by `id` asc limit ?", table, predicate)
	return query, []any{value, int64(f.Limit) + 1}, nil
}
func (m *customUsersModel) SearchReferences(ctx context.Context, f UserReferenceFilter) ([]UserReferenceRow, error) {
	query, args, err := referenceQuery(m.table, f)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, readquery.Timeout)
	defer cancel()
	var rows []UserReferenceRow
	if err := m.QueryRowsNoCacheCtx(ctx, &rows, query, args...); err != nil {
		return nil, err
	}
	return rows, nil
}
