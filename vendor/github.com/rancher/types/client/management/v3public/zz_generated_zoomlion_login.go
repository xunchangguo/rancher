package client

const (
	ZoomlionLoginType              = "zoomlionLogin"
	ZoomlionLoginFieldCode         = "code"
	ZoomlionLoginFieldDescription  = "description"
	ZoomlionLoginFieldResponseType = "responseType"
	ZoomlionLoginFieldTTLMillis    = "ttl"
)

type ZoomlionLogin struct {
	Code         string `json:"code,omitempty" yaml:"code,omitempty"`
	Description  string `json:"description,omitempty" yaml:"description,omitempty"`
	ResponseType string `json:"responseType,omitempty" yaml:"responseType,omitempty"`
	TTLMillis    int64  `json:"ttl,omitempty" yaml:"ttl,omitempty"`
}
