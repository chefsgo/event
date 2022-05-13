package event

import (
	"fmt"
	"sync"

	. "github.com/chefsgo/base"
	"github.com/chefsgo/chef"
	"github.com/chefsgo/util"
)

func init() {
	chef.Register(module)
}

var (
	module = &Module{
		configs:   make(map[string]Config, 0),
		drivers:   make(map[string]Driver, 0),
		instances: make(map[string]Instance, 0),

		events:   make(map[string]Event, 0),
		notices:  make(map[string]Notice, 0),
		filters:  make(map[string]Filter, 0),
		handlers: make(map[string]Handler, 0),

		relates: make(map[string]string, 0),
	}
)

type (
	Module struct {
		mutex sync.Mutex

		connected, initialized, launched bool

		configs map[string]Config
		drivers map[string]Driver

		events   map[string]Event
		notices  map[string]Notice
		filters  map[string]Filter
		handlers map[string]Handler

		relates map[string]string

		requestFilters  []ctxFunc
		executeFilters  []ctxFunc
		responseFilters []ctxFunc

		foundHandlers  []ctxFunc
		errorHandlers  []ctxFunc
		failedHandlers []ctxFunc
		deniedHandlers []ctxFunc

		instances map[string]Instance

		weights  map[string]int
		hashring *util.HashRing
	}

	Configs map[string]Config
	Config  struct {
		Driver  string
		Codec   string
		Weight  int
		Prefix  string
		Setting Map
	}
	Instance struct {
		module  *Module
		name    string
		config  Config
		connect Connect
	}
)

// Driver 注册驱动
func (module *Module) Driver(name string, driver Driver, override bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	if driver == nil {
		panic("Invalid event driver: " + name)
	}

	if override {
		module.drivers[name] = driver
	} else {
		if module.drivers[name] == nil {
			module.drivers[name] = driver
		}
	}
}

func (this *Module) Config(name string, config Config, override bool) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if name == "" {
		name = chef.DEFAULT
	}

	if override {
		this.configs[name] = config
	} else {
		if _, ok := this.configs[name]; ok == false {
			this.configs[name] = config
		}
	}
}
func (this *Module) Configs(config Configs, override bool) {
	for key, val := range config {
		this.Config(key, val, override)
	}
}

// Publish 发起消息
func (this *Module) publishTo(connect, name string, meta chef.Metadata) error {
	locate := module.hashring.Locate(name)

	if inst, ok := module.instances[locate]; ok {

		// 看看是不是有 notice 定义，如果有，并有args定义，要处理参数包装
		if meta.Payload != nil {
			if notice, ok := this.notices[name]; ok {
				if notice.Args != nil {
					value := Map{}
					res := chef.Mapping(notice.Args, meta.Payload, value, notice.Nullable, false)
					if res == nil || res.OK() {
						meta.Payload = value
					}
				}
			}
		}

		//原数据
		bytes, err := chef.Marshal(inst.config.Codec, &meta)
		if err != nil {
			return err
		}

		name := inst.config.Prefix + name
		return inst.connect.Publish(name, bytes)
	}

	return errInvalidConnection
}

func (this *Module) relateKey(conn, alias string) string {
	return fmt.Sprintf("%s-%s")
}

func (this *Module) publish(name string, meta chef.Metadata) error {
	locate := module.hashring.Locate(name)
	return this.publishTo(locate, name, meta)
}
