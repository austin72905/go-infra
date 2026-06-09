package module

/*
原本叫Config , 現在改叫Component
本就是拿來裝 framework 建立過的各種 config instance。

也就是像這些東西都會放進去：

SchedulerConfig
KafkaConfig
HTTPConfig
GrpcServerConfig
RedisConfig
MongoConfig
DBConfig
APIConfig
其他 Common 暴露出來的 XxxConfig
*/
type ComponentRegistry struct {
	components map[string]Component // go 不會返回 *Component
}

func NewComponentRegistry() *ComponentRegistry {
	return &ComponentRegistry{
		components: make(map[string]Component),
	}
}

func (cr *ComponentRegistry) Register(name string, component Component) {
	cr.components[name] = component
}

// 檢查是否有註冊過Component
func (cr *ComponentRegistry) LookupComponent(name string) (Component, bool) {
	component, ok := cr.components[name]
	return component, ok
}

func (cr *ComponentRegistry) GetComponent(name string) Component {
	return cr.components[name]
}
