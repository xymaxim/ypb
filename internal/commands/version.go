package commands

import (
	"fmt"

	versionpkg "github.com/xymaxim/ypb/internal/version"
)

type Version struct {
	Short bool `help:"Show only the version number" short:"s"`
}

func (c *Version) Run() error {
	var version string
	if c.Short {
		version = versionpkg.GetShort()
	} else {
		version = versionpkg.GetFull()
	}

	fmt.Println(version)

	return nil
}
