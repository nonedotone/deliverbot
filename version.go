package main

import (
	"fmt"
	"runtime"
)

var (
	Commit = "unknown"
)

func Version() string {
	if Commit != "unknown" && len(Commit) > 12 {
		Commit = Commit[:11]
	}
	buildOS := runtime.Version() + "/" + runtime.GOOS + "/" + runtime.GOARCH
	return fmt.Sprintf("gitCommit:%s\ngoVerson:%s", Commit, buildOS)
}
