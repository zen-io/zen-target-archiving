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
	Name          string            `mapstructure:"name" zen:"yes" desc:"Name for the target"`
	Description   string            `mapstructure:"desc" zen:"yes" desc:"Target description"`
	Labels        []string          `mapstructure:"labels" zen:"yes" desc:"Labels to apply to the targets"` //
	Deps          []string          `mapstructure:"deps" zen:"yes" desc:"Build dependencies"`
	PassEnv       []string          `mapstructure:"pass_env" zen:"yes" desc:"List of environment variable names that will be passed from the OS environment, they are part of the target hash"`
	PassSecretEnv []string          `mapstructure:"pass_secret_env" zen:"yes" desc:"List of environment variable names that will be passed from the OS environment, they are not used to calculate the target hash"`
	Env           map[string]string `mapstructure:"env" zen:"yes" desc:"Key-Value map of static environment variables to be used"`
	Visibility    []string          `mapstructure:"visibility" zen:"yes" desc:"List of visibility for this target"`
	Srcs          []string          `mapstructure:"srcs"`
	Type          *ArchiveType      `mapstructure:"type"`
	Out           string            `mapstructure:"out"`
	ExclusionFile *string           `mapstructure:"exclusion_file"`
	Exclusions    []string          `mapstructure:"exclusions"`
}

func (ac ArchiveConfig) GetTargets(_ *zen_targets.TargetConfigContext) ([]*zen_targets.TargetBuilder, error) {
	srcs := map[string][]string{"srcs": ac.Srcs}
	if ac.ExclusionFile != nil {
		srcs["exclusion"] = []string{*ac.ExclusionFile}
	}

	tb := zen_targets.ToTarget(ac)
	tb.Srcs = srcs
	tb.Outs = []string{ac.Out}
	tb.Scripts["build"] = &zen_targets.TargetBuilderScript{
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
				return fmt.Errorf("archive type %v not accepted. Valid choices are 'zip' and 'tar'", ac.Type)
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
	}

	return []*zen_targets.TargetBuilder{tb}, nil
}
