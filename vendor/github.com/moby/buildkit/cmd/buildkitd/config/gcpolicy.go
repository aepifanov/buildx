package config

import (
	"encoding"
	"strconv"
	"strings"
	"time"

	"github.com/docker/go-units"
	"github.com/moby/buildkit/util/bklog"
	"github.com/moby/buildkit/util/disk"
	"github.com/pkg/errors"
)

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalText(textb []byte) error {
	text := stripQuotes(string(textb))
	if len(text) == 0 {
		return nil
	}

	if duration, err := time.ParseDuration(text); err == nil {
		d.Duration = duration
		return nil
	}

	if i, err := strconv.ParseInt(text, 10, 64); err == nil {
		d.Duration = time.Duration(i) * time.Second
		return nil
	}

	return errors.Errorf("invalid duration %s", text)
}

var _ encoding.TextUnmarshaler = &Duration{}

type DiskSpace struct {
	Bytes      int64
	Percentage int64
}

var _ encoding.TextUnmarshaler = &DiskSpace{}

func (d *DiskSpace) UnmarshalText(textb []byte) error {
	text := stripQuotes(string(textb))
	if len(text) == 0 {
		return nil
	}

	if text2 := strings.TrimSuffix(text, "%"); len(text2) < len(text) {
		i, err := strconv.ParseInt(text2, 10, 64)
		if err != nil {
			return err
		}
		d.Percentage = i
		return nil
	}

	if i, err := units.RAMInBytes(text); err == nil {
		d.Bytes = i
		return nil
	}

	return errors.Errorf("invalid disk space %s", text)
}

const defaultCap int64 = 2e9 // 2GB

func DefaultGCPolicy(keep DiskSpace) []GCPolicy {
	if keep == (DiskSpace{}) {
		keep = DetectDefaultGCCap()
	}
	return []GCPolicy{
		// if build cache uses more than 512MB delete the most easily reproducible data after it has not been used for 2 days
		{
			Filters:      []string{"type==source.local,type==exec.cachemount,type==source.git.checkout"},
			KeepDuration: Duration{Duration: time.Duration(48) * time.Hour}, // 48h
			MaxStorage:   DiskSpace{Bytes: 512 * 1e6},                       // 512MB
		},
		// remove any data not used for 60 days
		{
			KeepDuration: Duration{Duration: time.Duration(60) * 24 * time.Hour}, // 60d
			MaxStorage:   keep,
		},
		// keep the unshared build cache under cap
		{
			MaxStorage: keep,
		},
		// if previous policies were insufficient start deleting internal data to keep build cache under cap
		{
			All:        true,
			MaxStorage: keep,
		},
	}
}

func stripQuotes(s string) string {
	if len(s) == 0 {
		return s
	}
	if s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func DetectDefaultGCCap() DiskSpace {
	return DiskSpace{Percentage: DiskSpacePercentage}
}

func (d DiskSpace) AsBytes(root string) int64 {
	if d.Bytes != 0 {
		return d.Bytes
	}
	if d.Percentage == 0 {
		return 0
	}

	dstat, err := disk.GetDiskStat(root)
	if err != nil {
		bklog.L.Warnf("failed to get disk size: %v", err)
		return defaultCap
	}
	avail := dstat.Total * d.Percentage / 100
	rounded := (avail/(1<<30) + 1) * 1e9 // round up
	return rounded
}
