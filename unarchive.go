package archiving

import (
	"fmt"
	"strings"

	zen_targets "github.com/zen-io/zen-core/target"
)

type UnarchiveConfig struct {
	Name          string            `mapstructure:"name" zen:"yes" desc:"Name for the target"`
	Description   string            `mapstructure:"desc" zen:"yes" desc:"Target description"`
	Labels        []string          `mapstructure:"labels" zen:"yes" desc:"Labels to apply to the targets"`
	Deps          []string          `mapstructure:"deps" zen:"yes" desc:"Build dependencies"`
	PassEnv       []string          `mapstructure:"pass_env" zen:"yes" desc:"List of environment variable names that will be passed from the OS environment, they are part of the target hash"`
	PassSecretEnv []string          `mapstructure:"pass_secret_env" zen:"yes" desc:"List of environment variable names that will be passed from the OS environment, they are not used to calculate the target hash"`
	Env           map[string]string `mapstructure:"env" zen:"yes" desc:"Key-Value map of static environment variables to be used"`
	Visibility    []string          `mapstructure:"visibility" zen:"yes" desc:"List of visibility for this target"`
	Src           string            `mapstructure:"src"`
	ExportedFiles []string          `mapstructure:"exported_files"`
	Binary        bool              `mapstructure:"binary"`
}

func (uc UnarchiveConfig) GetTargets(tcc *zen_targets.TargetConfigContext) ([]*zen_targets.TargetBuilder, error) {
	var outs []string

	if uc.ExportedFiles != nil {
		outs = uc.ExportedFiles
	} else {
		outs = append(outs, "**/*")
	}

	tb := zen_targets.ToTarget(uc)
	tb.Srcs = map[string][]string{"src": {uc.Src}}
	tb.Outs = outs

	tb.Scripts["build"] = &zen_targets.TargetBuilderScript{
		Deps: uc.Deps,
		Run: func(target *zen_targets.Target, runCtx *zen_targets.RuntimeContext) error {
			if len(target.Srcs["src"]) == 0 {
				return fmt.Errorf("no srcs provided")
			}
			filePath := target.Srcs["src"][0]

			var decompressFunc func(string, string) ([]string, error)
			switch {
			case strings.HasSuffix(filePath, ".zip"):
				decompressFunc = ExtractZip
			case strings.HasSuffix(filePath, ".tar"):
				decompressFunc = ExtractTar
			case strings.HasSuffix(filePath, ".tar.gz") || strings.HasSuffix(filePath, ".tgz"):
				decompressFunc = ExtractTarGz
			default:
				return fmt.Errorf("unknown file type")
			}

			_, err := decompressFunc(filePath, target.Cwd)
			if err != nil {
				return fmt.Errorf("error decompressing file: %w", err)
			}

			return nil
		},
	}

	if uc.Binary {
		tb.Binary = true
	}

	return []*zen_targets.TargetBuilder{tb}, nil
}
