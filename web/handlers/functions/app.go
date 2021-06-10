package functions

import (
	"github.com/jenkins-x-plugins/lighthouse-webui-plugin/internal/version"
)

func AppVersion() string {
	return version.Version
}
