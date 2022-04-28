package event

import (
	. "github.com/chefsgo/base"
	"github.com/chefsgo/chef"
)

// . "github.com/chefsgo/base"

// Publish 发起消息
func (this *Module) Publish(name string, values ...Map) error {
	//原数据
	var payload Map
	if len(values) > 0 {
		payload = values[0]
	}
	meta := chef.Metadata{Name: name, Payload: payload}

	return this.publish(name, meta)
}
