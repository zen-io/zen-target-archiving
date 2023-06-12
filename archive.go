package archiving

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	ahoy_targets "gitlab.com/hidothealth/platform/ahoy/src/target"
	"gitlab.com/hidothealth/platform/ahoy/src/utils"

	"golang.org/x/exp/slices"
)

type ArchiveType string

const (
	Zip ArchiveType = "zip"
	Tar ArchiveType = "tar"
)

type Archiver interface {
	CompressFile(src, dest string, info fs.FileInfo) error
	Finish()
}

type ArchiveConfig struct {
	ahoy_targets.BaseFields `mapstructure:",squash"`
	Srcs                    []string    `mapstructure:"srcs"`
	Type                    ArchiveType `mapstructure:"type"`
	Out                     string      `mapstructure:"out"`
	ExclusionFile           *string     `mapstructure:"exclusion_file"`
	Exclusions              []string    `mapstructure:"exclusions"`
}

func (ac ArchiveConfig) GetTargets(_ *ahoy_targets.TargetConfigContext) ([]*ahoy_targets.Target, error) {
	srcs := map[string][]string{"srcs": ac.Srcs}
	if ac.ExclusionFile != nil {
		srcs["exclusion"] = []string{*ac.ExclusionFile}
	}

	opts := []ahoy_targets.TargetOption{
		ahoy_targets.WithSrcs(srcs),
		ahoy_targets.WithOuts([]string{ac.Out}),
		ahoy_targets.WithLabels(ac.Labels),
	}

	opts = append(opts,
		ahoy_targets.WithTargetScript("build", &ahoy_targets.TargetScript{
			Deps: ac.Deps,
			Run: func(target *ahoy_targets.Target, runCtx *ahoy_targets.RuntimeContext) error {
				var archiver Archiver
				var err error
				out := filepath.Join(target.Cwd, ac.Out)

				switch ac.Type {
				case Zip:
					archiver, err = NewZipArchive(out)
				case Tar:
					archiver, err = NewTarArchive(out)
				default:
					return fmt.Errorf("archive type %s not accepted. Valid choices are 'zip' and 'tar'", ac.Type)
				}
				if err != nil {
					return err
				}

				exclusions := make([]string, 0)
				if ac.Exclusions != nil {
					exclusions = append(exclusions, ac.Exclusions...)
				}
				if ac.ExclusionFile != nil {
					exclusionsInFile, err := utils.ReadExclusionFile(*ac.ExclusionFile)
					if err != nil {
						return fmt.Errorf("loading exclusions from file: %w", err)
					}
					exclusions = append(exclusions, exclusionsInFile...)
				}

				for _, srcs := range target.Srcs {
					for _, rootSrc := range srcs {
						if err := filepath.Walk(rootSrc, func(from string, info fs.FileInfo, err error) error {
							if err != nil {
								return err
							}

							if slices.ContainsFunc(exclusions, func(item string) bool {
								return strings.Contains(from, item)
							}) {
								return nil
							}

							to := strings.TrimPrefix(from, target.Cwd+"/")
							if err = archiver.CompressFile(from, to, info); err != nil {
								return err
							}

							return nil
						}); err != nil {
							return err
						}
					}
				}
				archiver.Finish()

				return nil
			},
		}),
	)

	return []*ahoy_targets.Target{
		ahoy_targets.NewTarget(
			ac.Name,
			opts...,
		),
	}, nil
}
