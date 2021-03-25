module github.com/joelanford/channelizer

go 1.16

require (
	github.com/blang/semver/v4 v4.0.0
	github.com/operator-framework/operator-registry v1.16.2-0.20210323200419-4b5d403b8b91
	github.com/spf13/cobra v1.1.1
)

replace github.com/operator-framework/operator-registry => ../../operator-framework/operator-registry
