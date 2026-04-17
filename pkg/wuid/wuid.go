package wuid

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"sync"

	"github.com/edwingeng/wuid/mysql/wuid"
)

var w *wuid.WUID

func Init(dsn string) {
	newDB := func() (*sql.DB, bool, error) {
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return nil, false, err
		}
		return db, true, nil
	}
	w = wuid.NewWUID("default", nil)
	_ = w.LoadH28FromMysql(newDB, "wuid")
}

var (
	once sync.Once
)

func GenUid(dsn string) string {
	// once.Do 保证了无论多少个并发请求砸过来，Init(dsn) 都只会被执行一次
	// 并且其他并发请求会在这里阻塞，直到 Init 执行完毕才会往下走
	once.Do(func() {
		Init(dsn)
	})

	return fmt.Sprintf("%#016x", w.Next())
}

func CombineId(aid, bid string) string {
	ids := []string{aid, bid}

	sort.Slice(ids, func(i, j int) bool {
		a, _ := strconv.ParseUint(ids[i], 0, 64)
		b, _ := strconv.ParseUint(ids[j], 0, 64)
		return a < b
	})

	return fmt.Sprintf("%s_%s", ids[0], ids[1])
}
