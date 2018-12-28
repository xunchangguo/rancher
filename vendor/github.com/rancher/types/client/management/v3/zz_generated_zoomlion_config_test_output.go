package client

const (
	ZoomlionConfigTestOutputType             = "zoomlionConfigTestOutput"
	ZoomlionConfigTestOutputFieldRedirectURL = "redirectUrl"
)

type ZoomlionConfigTestOutput struct {
	RedirectURL string `json:"redirectUrl,omitempty" yaml:"redirectUrl,omitempty"`
}
