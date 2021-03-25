package channelizer

import (
	"fmt"
	"log"
	"sort"

	"github.com/blang/semver/v4"
	"github.com/operator-framework/operator-registry/pkg/declcfg"
	"github.com/operator-framework/operator-registry/pkg/property"
)

type Channelizer interface {
	Channelize([]declcfg.Bundle) (string, error)
}

type None struct{}

func (c None) Channelize(_ []declcfg.Bundle) (string, error) {
	return "", nil
}

type Semver struct {
	CombinePreReleases   bool
	ConsiderBuildID      bool
	ConnectMinorChannels bool
}

func (c Semver) Channelize(in []*declcfg.Bundle) (string, error) {
	versions := map[string]semver.Version{}
	for _, b := range in {
		props, err := property.Parse(b.Properties)
		if err != nil {
			return "", fmt.Errorf("parse properties for bundle %q: %v", b.Name, err)
		}
		version := ""
		if len(props.Packages) == 0 {
			return "", fmt.Errorf("could not determine version for bundle %q: no olm.package property", b.Name)
		}
		version = props.Packages[0].Version
		v, err := semver.ParseTolerant(version)
		if err != nil {
			return "", fmt.Errorf("parse semver for bundle %q: %v", b.Name, err)
		}
		versions[b.Name] = v
	}

	sort.Slice(in, func(i, j int) bool {
		vi, vj := versions[in[i].Name], versions[in[j].Name]
		switch vi.Compare(vj) {
		case -1:
			return false
		case 1:
			return true
		}
		if !c.ConsiderBuildID {
			return true
		}
		return !sliceIsLess(vi.Build, vj.Build)
	})

	defaultChannel := ""
	for i, b := range in {
		b.Properties = removeUpgradeEdgeProperties(b.Properties)
		var (
			nb *declcfg.Bundle
			nv *semver.Version
		)
		if i+1 < len(in) {
			nextb := in[i+1]
			nextv := versions[nextb.Name]
			nb = nextb
			nv = &nextv
		}
		bv := versions[b.Name]
		if i == 0 {
			defaultChannel = fmt.Sprintf("v%d", bv.Major)
		}
		isPreRelease := len(bv.Pre) > 0
		channels := []property.Channel{}
		if isPreRelease && !c.CombinePreReleases {
			replaces := ""
			if nv != nil && nv.Major == bv.Major && nv.Minor == bv.Minor && len(nv.Pre) > 0 {
				replaces = nb.Name
			}
			channels = append(channels, property.Channel{
				Name:     fmt.Sprintf("pre-v%d.%d", bv.Major, bv.Minor),
				Replaces: replaces,
			})
		} else {
			replaces := ""
			if nv != nil && nv.Major == bv.Major && (len(nv.Pre) == 0 || c.CombinePreReleases) {
				replaces = nb.Name
			}
			channels = append(channels, property.Channel{
				Name:     fmt.Sprintf("v%d", bv.Major),
				Replaces: replaces,
			})
			replaces = ""
			if nv != nil && nv.Major == bv.Major && nv.Minor == bv.Minor && (len(nv.Pre) == 0 || c.CombinePreReleases) {
				replaces = nb.Name
			}
			channels = append(channels, property.Channel{
				Name:     fmt.Sprintf("v%d.%d", bv.Major, bv.Minor),
				Replaces: replaces,
			})
		}
		if c.ConnectMinorChannels {
			if nv != nil && nv.Major == bv.Major && nv.Minor != bv.Minor {
				log.Printf("Bundle %q skips %q", b.Name, nb.Name)
				b.Properties = append(b.Properties, property.MustBuildSkips(nb.Name))
			}
		}
		for _, ch := range channels {
			if ch.Replaces == "" {
				log.Printf("Bundle %q replaces nothing in channel %q", b.Name, ch.Name)
			} else {
				log.Printf("Bundle %q replaces %q in channel %q", b.Name, ch.Replaces, ch.Name)
			}
			b.Properties = append(b.Properties, property.MustBuild(&ch))
		}
	}
	return defaultChannel, nil
}

func removeUpgradeEdgeProperties(in []property.Property) []property.Property {
	out := []property.Property{}
	for _, x := range in {
		switch x.Type {
		case property.TypeChannel, property.TypeSkips, property.TypeSkipRange:
		default:
			out = append(out, x)
		}
	}
	return out
}

func sliceIsLess(s1, s2 []string) bool {
	for {
		if len(s1) == 0 {
			return len(s2) > 0
		} else if len(s2) == 0 {
			return !(len(s1) > 0)
		} else if s1[0] != s2[0] {
			return s1[0] < s2[0]
		}
		s1, s2 = s1[1:], s2[1:]
	}
}
