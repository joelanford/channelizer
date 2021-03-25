package main

import (
	"log"
	"os"

	"github.com/joelanford/channelizer/pkg/channelizer"
	"github.com/operator-framework/operator-registry/pkg/declcfg"
	"github.com/spf13/cobra"
)

func main() {
	cmd := cobra.Command{
		Use:   "channelizer <declarative_configs> <package_name>",
		Short: "Rebuild channels for a package based on semver",
		Long: `
Rebuild channels for a package based on semver.

channelizer will completely rewrite channels for a particular
package based on semver. It will create channels for all major and
major.minor versions in a package.
`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			configs := args[0]
			pkgName := args[1]

			if err := run(configs, pkgName); err != nil {
				log.Fatal(err)
			}
		},
	}
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func run(configs string, pkgName string) error {
	cfg, err := declcfg.LoadDir(configs)
	if err != nil {
		return err
	}

	pkgBundles := []*declcfg.Bundle{}
	for i, b := range cfg.Bundles {
		if pkgName == b.Package {
			pkgBundles = append(pkgBundles, &cfg.Bundles[i])
		}
	}

	c := &channelizer.Semver{
		CombinePreReleases:   true,
		ConsiderBuildID:      true,
		ConnectMinorChannels: true,
	}
	defaultChannel, err := c.Channelize(pkgBundles)
	if err != nil {
		return err
	}
	if defaultChannel != "" {
		for i, p := range cfg.Packages {
			if pkgName == p.Name {
				log.Printf("Setting default channel %q for package %q", defaultChannel, pkgName)
				cfg.Packages[i].DefaultChannel = defaultChannel
			}
		}
	}
	if _, err := declcfg.ConvertToModel(*cfg); err != nil {
		return err
	}
	if err := os.RemoveAll(configs); err != nil {
		return err
	}
	return declcfg.WriteDir(*cfg, configs)
}
