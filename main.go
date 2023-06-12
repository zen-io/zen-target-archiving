package archiving

import (
	ahoy_targets "gitlab.com/hidothealth/platform/ahoy/src/target"
)

var KnownTargets = ahoy_targets.TargetCreatorMap{
	"unarchive": UnarchiveConfig{},
	"archive":   ArchiveConfig{},
}
