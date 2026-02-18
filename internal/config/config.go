package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
)

const (
	dirName      = ".locwp"
	portStart    = 8081
	portEnd      = 8180
)

// BaseDir returns the locwp data directory, creating it if needed.
// Honors LOCWP_HOME env var, defaults to ~/.locwp.
func BaseDir() string {
	dir := os.Getenv("LOCWP_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, dirName)
	}
	os.MkdirAll(dir, 0755)
	return dir
}

// NextPort finds the next available port in the range.
func NextPort(baseDir string) (int, error) {
	for p := portStart; p <= portEnd; p++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if err == nil {
			ln.Close()
			return p, nil
		}
	}
	return 0, fmt.Errorf("no available port in range %d-%d", portStart, portEnd)
}
