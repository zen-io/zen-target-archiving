package archiving

import (
	"fmt"
	"strings"

	zen_targets "github.com/zen-io/zen-core/target"
)

type UnarchiveConfig struct {
	Name          string            `mapstructure:"name" desc:"Name for the target"`
	Description   string            `mapstructure:"desc" desc:"Target description"`
	Labels        []string          `mapstructure:"labels" desc:"Labels to apply to the targets"`
	Deps          []string          `mapstructure:"deps" desc:"Build dependencies"`
	PassEnv       []string          `mapstructure:"pass_env" desc:"List of environment variable names that will be passed from the OS environment, they are part of the target hash"`
	SecretEnv     []string          `mapstructure:"secret_env" desc:"List of environment variable names that will be passed from the OS environment, they are not used to calculate the target hash"`
	Env           map[string]string `mapstructure:"env" desc:"Key-Value map of static environment variables to be used"`
	Visibility    []string          `mapstructure:"visibility" desc:"List of visibility for this target"`
	Src           string            `mapstructure:"src"`
	ExportedFiles []string          `mapstructure:"exported_files"`
	Binary        bool              `mapstructure:"binary"`
}

func (uc UnarchiveConfig) GetTargets(tcc *zen_targets.TargetConfigContext) ([]*zen_targets.Target, error) {
	var outs []string

	if uc.ExportedFiles != nil {
		outs = uc.ExportedFiles
	} else {
		outs = append(outs, "**/*")
	}

	opts := []zen_targets.TargetOption{
		zen_targets.WithSrcs(map[string][]string{"src": {uc.Src}}),
		zen_targets.WithOuts(outs),
		zen_targets.WithLabels(uc.Labels),
		zen_targets.WithVisibility(uc.Visibility),
		zen_targets.WithPassEnv(uc.PassEnv),
		zen_targets.WithSecretEnvVars(uc.SecretEnv),
		zen_targets.WithDescription(uc.Description),
		zen_targets.WithTargetScript("build", &zen_targets.TargetScript{
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
		}),
	}

	if uc.Binary {
		opts = append(opts, zen_targets.WithBinary())
	}

	return []*zen_targets.Target{
		zen_targets.NewTarget(
			uc.Name,
			opts...,
		),
	}, nil
}
