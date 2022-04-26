package event

type (
	// Driver 数据驱动
	Driver interface {
		Connect(name string, config Config) (Connect, error)
	}
	Health struct {
		Workload int64
	}

	Delegate interface {
		Serve(string, []byte)
	}

	// Connect 连接
	Connect interface {
		Open() error
		Health() (Health, error)
		Close() error

		Accept(Delegate) error
		Register(group, name string) error

		Start() error

		Publish(name string, data []byte) error
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
