//go:build darwin

package main

import (
	"os"
	"syscall"
	"time"
)

// getBirthTime はmacOSでファイルの作成日時を取得します。取得できない場合は更新日時を返します。
func getBirthTime(info os.FileInfo) time.Time {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec)
	}
	return info.ModTime()
}
