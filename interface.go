package event

import (
	. "github.com/chefsgo/base"
	"github.com/chefsgo/chef"
	"github.com/chefsgo/util"
)

func (this *Module) Register(name string, value Any, override bool) {
	switch config := value.(type) {
	case Driver:
		this.Driver(name, config, override)
	case Event:
		this.Event(name, config, override)
	case Notice:
		this.Notice(name, config, override)
	case Filter:
		this.Filter(name, config, override)
	case Handler:
		this.Handler(name, config, override)
	}
}

func (this *Module) configure(name string, config Map) {
	cfg := Config{
		Driver: chef.DEFAULT, Weight: 1, Codec: chef.JSON,
	}
	//如果已经存在了，用现成的改写
	if vv, ok := this.configs[name]; ok {
		cfg = vv
	}

	if driver, ok := config["driver"].(string); ok {
		cfg.Driver = driver
	}

	//分配权重
	if weight, ok := config["weight"].(int); ok {
		cfg.Weight = weight
	}
	if weight, ok := config["weight"].(int64); ok {
		cfg.Weight = int(weight)
	}
	if weight, ok := config["weight"].(float64); ok {
		cfg.Weight = int(weight)
	}

	if weight, ok := config["weight"].(float64); ok {
		cfg.Weight = int(weight)
	}

	if setting, ok := config["setting"].(Map); ok {
		cfg.Setting = setting
	}

	//保存配置
	this.configs[name] = cfg
}
func (this *Module) Configure(value Any) {
	if cfg, ok := value.(Config); ok {
		this.configs[chef.DEFAULT] = cfg
		return
	}
	if cfg, ok := value.(map[string]Config); ok {
		this.configs = cfg
		return
	}

	var global Map
	if cfg, ok := value.(Map); ok {
		global = cfg
	} else {
		return
	}

	var config Map
	if vvv, ok := global["event"].(Map); ok {
		config = vvv
	}

	//记录上一层的配置，如果有的话
	defConfig := Map{}

	for key, val := range config {
		if conf, ok := val.(Map); ok {
			//直接注册，然后删除当前key
			this.configure(key, conf)
		} else {
			//记录上一层的配置，如果有的话
			defConfig[key] = val
		}
	}

	if len(defConfig) > 0 {
		this.configure(chef.DEFAULT, defConfig)
	}
}
func (this *Module) Initialize() {
	if this.initialized {
		return
	}

	// 如果没有配置任何连接时，默认一个
	if len(this.configs) == 0 {
		this.configs[chef.DEFAULT] = Config{
			Driver: chef.DEFAULT, Weight: 1, Codec: chef.JSON,
		}
	} else {
		// 默认分布， 如果想不参与分布，Weight设置为小于0 即可
		for key, config := range this.configs {
			if config.Weight == 0 {
				config.Weight = 1
			}
			this.configs[key] = config
		}

	}

	//拦截器
	this.serveFilters = make([]ctxFunc, 0)
	this.requestFilters = make([]ctxFunc, 0)
	this.executeFilters = make([]ctxFunc, 0)
	this.responseFilters = make([]ctxFunc, 0)
	for _, filter := range this.filters {
		if filter.Serve != nil {
			this.serveFilters = append(this.serveFilters, filter.Serve)
		}
		if filter.Request != nil {
			this.requestFilters = append(this.requestFilters, filter.Request)
		}
		if filter.Execute != nil {
			this.executeFilters = append(this.executeFilters, filter.Execute)
		}
		if filter.Response != nil {
			this.responseFilters = append(this.responseFilters, filter.Response)
		}
	}

	//处理器
	this.foundHandlers = make([]ctxFunc, 0)
	this.errorHandlers = make([]ctxFunc, 0)
	this.failedHandlers = make([]ctxFunc, 0)
	this.deniedHandlers = make([]ctxFunc, 0)
	for _, filter := range this.handlers {
		if filter.Found != nil {
			this.foundHandlers = append(this.foundHandlers, filter.Found)
		}
		if filter.Error != nil {
			this.errorHandlers = append(this.errorHandlers, filter.Error)
		}
		if filter.Failed != nil {
			this.failedHandlers = append(this.failedHandlers, filter.Failed)
		}
		if filter.Denied != nil {
			this.deniedHandlers = append(this.deniedHandlers, filter.Denied)
		}
	}

	this.initialized = true
}
func (this *Module) Connect() {
	if this.connected {
		return
	}

	//记录要参与分布的连接和权重
	weights := make(map[string]int)

	for name, config := range this.configs {
		driver, ok := this.drivers[config.Driver]
		if ok == false {
			panic("Invalid event driver: " + config.Driver)
		}

		// 建立连接
		connect, err := driver.Connect(name, config)
		if err != nil {
			panic("Failed to connect to event: " + err.Error())
		}

		// 打开连接
		err = connect.Open()
		if err != nil {
			panic("Failed to open event connect: " + err.Error())
		}

		inst := Instance{
			this, name, config, connect,
		}

		// 指定委托
		connect.Accept(&inst)

		//注册事件
		for msgName, msgConfig := range this.events {
			if msgConfig.Connect == "" || msgConfig.Connect == "*" || msgConfig.Connect == name {
				for _, alias := range msgConfig.Alias {
					// 注册事件
					if err := connect.Register(msgConfig.Group, alias); err != nil {
						panic("Failed to register event: " + err.Error())
					}
					// 记录关联 conn-alias -> event
					relate := this.relateKey(name, alias)
					this.relates[relate] = msgName
				}
			}
		}

		//保存实例
		this.instances[name] = inst

		//只有设置了权重的才参与分布
		if config.Weight > 0 {
			weights[name] = config.Weight
		}
	}

	//hashring分片
	this.weights = weights
	this.hashring = util.NewHashRing(weights)

	this.connected = true
}
func (this *Module) Launch() {
	if this.launched {
		return
	}

	//全部开始来来来
	for _, inst := range this.instances {
		inst.connect.Start()
	}

	this.launched = true
}
func (this *Module) Terminate() {
	for _, ins := range this.instances {
		ins.connect.Close()
	}

	this.launched = false
	this.connected = false
	this.initialized = false
}
