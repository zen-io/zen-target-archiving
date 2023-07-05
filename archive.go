package archiving

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	zen_targets "github.com/zen-io/zen-core/target"
	"github.com/zen-io/zen-core/utils"

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
	Name          string            `mapstructure:"name" desc:"Name for the target"`
	Description   string            `mapstructure:"desc" desc:"Target description"`
	Labels        []string          `mapstructure:"labels" desc:"Labels to apply to the targets"` //
	Deps          []string          `mapstructure:"deps" desc:"Build dependencies"`
	PassEnv       []string          `mapstructure:"pass_env" desc:"List of environment variable names that will be passed from the OS environment, they are part of the target hash"`
	SecretEnv     []string          `mapstructure:"secret_env" desc:"List of environment variable names that will be passed from the OS environment, they are not used to calculate the target hash"`
	Env           map[string]string `mapstructure:"env" desc:"Key-Value map of static environment variables to be used"`
	Tools         map[string]string `mapstructure:"tools" desc:"Key-Value map of tools to include when executing this target. Values can be references"`
	Visibility    []string          `mapstructure:"visibility" desc:"List of visibility for this target"`
	Srcs          []string          `mapstructure:"srcs"`
	Type          *ArchiveType      `mapstructure:"type"`
	Out           string            `mapstructure:"out"`
	ExclusionFile *string           `mapstructure:"exclusion_file"`
	Exclusions    []string          `mapstructure:"exclusions"`
}

func (ac ArchiveConfig) GetTargets(_ *zen_targets.TargetConfigContext) ([]*zen_targets.Target, error) {
	srcs := map[string][]string{"srcs": ac.Srcs}
	if ac.ExclusionFile != nil {
		srcs["exclusion"] = []string{*ac.ExclusionFile}
	}

	opts := []zen_targets.TargetOption{
		zen_targets.WithSrcs(srcs),
		zen_targets.WithOuts([]string{ac.Out}),
		zen_targets.WithLabels(ac.Labels),
	}

	opts = append(opts,
		zen_targets.WithTargetScript("build", &zen_targets.TargetScript{
			Deps: ac.Deps,
			Run: func(target *zen_targets.Target, runCtx *zen_targets.RuntimeContext) error {
				var archiver Archiver
				var err error

				var outType ArchiveType
				if ac.Type == nil {
					outType = ArchiveType(utils.FileExtension(ac.Out)[1:])
				} else {
					outType = *ac.Type
				}
				out := filepath.Join(target.Cwd, target.Outs[0])
				switch outType {
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

	return []*zen_targets.Target{
		zen_targets.NewTarget(
			ac.Name,
			opts...,
		),
	}, nil
}
