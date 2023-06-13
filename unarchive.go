package archiving

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	zen_targets "github.com/zen-io/zen-core/target"
	"github.com/zen-io/zen-core/utils"

	doublestar "github.com/bmatcuk/doublestar/v4"
)

type UnarchiveConfig struct {
	zen_targets.BaseFields `mapstructure:",squash"`
	Src                    string   `mapstructure:"src"`
	Out                    *string  `mapstructure:"out"`
	ExportedFiles          []string `mapstructure:"exported_files"`
	Binary                 bool     `mapstructure:"binary"`
}

func (uc UnarchiveConfig) GetTargets(tcc *zen_targets.TargetConfigContext) ([]*zen_targets.Target, error) {
	var outs []string
	var extractTargetName string
	if len(uc.ExportedFiles) == 0 && uc.Out == nil {
		return nil, fmt.Errorf("need to provide either \"exported_files\" or \"out\"")
	} else if uc.Out != nil {
		if zen_targets.IsTargetReference(*uc.Out) {
			return nil, fmt.Errorf("out cannot be a reference")
		}
		extractTargetName = *uc.Out
		outs = []string{fmt.Sprintf("%s/**/*", *uc.Out)}
	} else {
		extractTargetName = "extract"
		outs = uc.ExportedFiles
	}

	opts := []zen_targets.TargetOption{
		zen_targets.WithSrcs(map[string][]string{"src": {uc.Src}}),
		zen_targets.WithOuts(outs),
		zen_targets.WithLabels(uc.Labels),
		zen_targets.WithTargetScript("build", &zen_targets.TargetScript{
			Deps: uc.Deps,
			Run: func(target *zen_targets.Target, runCtx *zen_targets.RuntimeContext) error {
				interpolatedTarget, err := target.Interpolate(extractTargetName)
				if err != nil {
					return err
				}

				extractTarget := filepath.Join(target.Cwd, interpolatedTarget)

				if len(target.Srcs["src"]) == 0 {
					return fmt.Errorf("no srcs provided")
				}
				filePath := target.Srcs["src"][0]

				if err := os.MkdirAll(extractTarget, os.ModePerm); err != nil {
					return err
				}
				var decompressFunc func(string, string) error
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

				// Decompress the file
				if err := decompressFunc(filePath, extractTarget); err != nil {
					return fmt.Errorf("error decompressing file: %w", err)
				}

				for _, o := range target.Outs {
					paths, err := doublestar.FilepathGlob(filepath.Join(extractTarget, o))
					if err != nil {
						return err
					}

					for _, from := range paths {
						to := filepath.Join(target.Cwd, strings.TrimPrefix(from, extractTarget))
						if err := utils.Copy(from, to); err != nil {
							return err
						}
					}
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
