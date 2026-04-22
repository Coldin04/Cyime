package utils

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetRunTimeCaller will return the current runner when Crafting this project, ONLY FOR DEV.
func GetRunTimeCallerPathDEV() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filename)
}

func GetRunTimeCallerPathPROD() string {
	cmd, _ := os.Getwd()
	return cmd
}

func GetRunTimeCallerPath() string {
	if os.Getenv("ENV") == "development" {
		return GetRunTimeCallerPathDEV()
	}
	return GetRunTimeCallerPathPROD()
}
