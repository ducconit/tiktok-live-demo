// Package build — thông tin build của binary (version, hash, date).
// Được inject qua ldflags lúc `make build` (xem Makefile + scripts/version.sh).
package build

import "fmt"

// Info — thông tin build, expose qua /healthz và log startup.
type Info struct {
	Version   string `json:"version"`
	BuildHash string `json:"build_hash"`
	BuildDate string `json:"build_date"`
}

// String — "1.0.0 (abc1234, 2026-08-11T...)".
func (i Info) String() string {
	return fmt.Sprintf("%s (%s, %s)", i.Version, i.BuildHash, i.BuildDate)
}
