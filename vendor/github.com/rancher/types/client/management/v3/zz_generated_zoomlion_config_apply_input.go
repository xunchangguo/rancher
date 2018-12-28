package client

const (
	ZoomlionConfigApplyInputType                = "zoomlionConfigApplyInput"
	ZoomlionConfigApplyInputFieldCode           = "code"
	ZoomlionConfigApplyInputFieldEnabled        = "enabled"
	ZoomlionConfigApplyInputFieldZoomlionConfig = "zoomlionConfig"
)

type ZoomlionConfigApplyInput struct {
	Code           string          `json:"code,omitempty" yaml:"code,omitempty"`
	Enabled        bool            `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	ZoomlionConfig *ZoomlionConfig `json:"zoomlionConfig,omitempty" yaml:"zoomlionConfig,omitempty"`
}
