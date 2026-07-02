//go:build !darwin

package main

import (
	"os"
	"time"
)

// getBirthTime はmacOS以外の環境でファイルの更新日時をフォールバックとして返します。
func getBirthTime(info os.FileInfo) time.Time {
	return info.ModTime()
}
