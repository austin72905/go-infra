package module

import (
	"embed"
	"fmt"
)

/*
服務可以用
a.Schedule()
a.Kafka()
a.Http()
a.Load(...)

// 非核心能力的例如scheduler 就可以用lazy create
*/
type ModuleBase struct {
	AppRuntime *AppRuntime
}

func (m *ModuleBase) SetAppRuntime(appRuntime *AppRuntime) {
	m.AppRuntime = appRuntime
}

func (m *ModuleBase) Web() *WebRuntime {
	return m.AppRuntime.Web
}

func (m *ModuleBase) Probe() *StartupProbe {
	return m.AppRuntime.Probe
}

func (m *ModuleBase) PropertyRuntime() *PropertyRuntime {
	return m.AppRuntime.Property
}

func (m *ModuleBase) Property(key string) string {
	return m.AppRuntime.Property.Property(key)
}

func (m *ModuleBase) RequiredProperty(key string) string {
	return m.AppRuntime.Property.RequiredProperty(key)
}

func (m *ModuleBase) LoadProperties(envFS map[string]embed.FS, env string, fileName string) error {
	return m.AppRuntime.Property.LoadProperties(envFS, env, fileName)
}

func (m *ModuleBase) Kafka(name ...string) *KafkaComponent {
	componentName := defaultKafkaComponentName
	if len(name) > 0 && name[0] != "" {
		componentName = kafkaComponentName(name[0])
	}

	if component, ok := m.AppRuntime.Components.LookupComponent(componentName); ok {
		kafkaComponent, typeOK := component.(*KafkaComponent)
		if !typeOK {
			panic(fmt.Sprintf("component %q is not a KafkaComponent", componentName))
		}
		return kafkaComponent
	}

	if m.AppRuntime.Lifecycle.Started {
		panic(fmt.Sprintf("%s component must be registered during Initialize()", componentName))
	}

	kafkaComponent := &KafkaComponent{}
	kafkaComponent.initialize(m.AppRuntime, componentName)
	m.AppRuntime.Components.Register(componentName, kafkaComponent)
	return kafkaComponent
}

func (m *ModuleBase) RabbitMQ(name ...string) *RabbitMQComponent {
	componentName := defaultRabbitMQComponentName
	if len(name) > 0 && name[0] != "" {
		componentName = rabbitMQComponentName(name[0])
	}

	if component, ok := m.AppRuntime.Components.LookupComponent(componentName); ok {
		rabbitMQComponent, typeOK := component.(*RabbitMQComponent)
		if !typeOK {
			panic(fmt.Sprintf("component %q is not a RabbitMQComponent", componentName))
		}
		return rabbitMQComponent
	}

	if m.AppRuntime.Lifecycle.Started {
		panic(fmt.Sprintf("%s component must be registered during Initialize()", componentName))
	}

	rabbitMQComponent := &RabbitMQComponent{}
	rabbitMQComponent.initialize(m.AppRuntime, componentName)
	m.AppRuntime.Components.Register(componentName, rabbitMQComponent)
	return rabbitMQComponent
}

func (m *ModuleBase) Mongo(name ...string) *MongoComponent {
	componentName := defaultMongoComponentName
	if len(name) > 0 && name[0] != "" {
		componentName = mongoComponentName(name[0])
	}

	if component, ok := m.AppRuntime.Components.LookupComponent(componentName); ok {
		mongoComponent, typeOK := component.(*MongoComponent)
		if !typeOK {
			panic(fmt.Sprintf("component %q is not a MongoComponent", componentName))
		}
		return mongoComponent
	}

	if m.AppRuntime.Lifecycle.Started {
		panic(fmt.Sprintf("%s component must be registered during Initialize()", componentName))
	}

	mongoComponent := &MongoComponent{}
	mongoComponent.initialize(m.AppRuntime, componentName)
	m.AppRuntime.Components.Register(componentName, mongoComponent)
	return mongoComponent
}

func (m *ModuleBase) Grpc() *GrpcComponent {
	if component, ok := m.AppRuntime.Components.LookupComponent(grpcComponentName); ok {
		grpcComponent, typeOK := component.(*GrpcComponent)
		if !typeOK {
			panic(fmt.Sprintf("component %q is not a GrpcComponent", grpcComponentName))
		}
		return grpcComponent
	}

	if m.AppRuntime.Lifecycle.Started {
		panic(fmt.Sprintf("%s component must be registered during Initialize()", grpcComponentName))
	}

	grpcComponent := &GrpcComponent{}
	grpcComponent.initialize(m.AppRuntime, grpcComponentName)
	m.AppRuntime.Components.Register(grpcComponentName, grpcComponent)
	return grpcComponent
}

func (m *ModuleBase) GrpcClient() *GrpcClientComponent {
	if component, ok := m.AppRuntime.Components.LookupComponent(grpcClientComponentName); ok {
		grpcClientComponent, typeOK := component.(*GrpcClientComponent)
		if !typeOK {
			panic(fmt.Sprintf("component %q is not a GrpcClientComponent", grpcClientComponentName))
		}
		return grpcClientComponent
	}

	if m.AppRuntime.Lifecycle.Started {
		panic(fmt.Sprintf("%s component must be registered during Initialize()", grpcClientComponentName))
	}

	grpcClientComponent := &GrpcClientComponent{}
	grpcClientComponent.initialize(m.AppRuntime, grpcClientComponentName)
	m.AppRuntime.Components.Register(grpcClientComponentName, grpcClientComponent)
	return grpcClientComponent
}

func (m *ModuleBase) DB(name ...string) *DBComponent {
	componentName := defaultDBComponentName
	if len(name) > 0 && name[0] != "" {
		componentName = dbComponentName(name[0])
	}

	if component, ok := m.AppRuntime.Components.LookupComponent(componentName); ok {
		dbComponent, typeOK := component.(*DBComponent)
		if !typeOK {
			panic(fmt.Sprintf("component %q is not a DBComponent", componentName))
		}
		return dbComponent
	}

	if m.AppRuntime.Lifecycle.Started {
		panic(fmt.Sprintf("%s component must be registered during Initialize()", componentName))
	}

	dbComponent := &DBComponent{}
	dbComponent.initialize(m.AppRuntime, componentName)
	m.AppRuntime.Components.Register(componentName, dbComponent)
	return dbComponent
}

func (m *ModuleBase) Redis(name ...string) *RedisComponent {
	componentName := defaultRedisComponentName
	if len(name) > 0 && name[0] != "" {
		componentName = redisNamedComponentName(name[0])
	}

	if component, ok := m.AppRuntime.Components.LookupComponent(componentName); ok {
		redisComponent, typeOK := component.(*RedisComponent)
		if !typeOK {
			panic(fmt.Sprintf("component %q is not a RedisComponent", componentName))
		}
		return redisComponent
	}

	if m.AppRuntime.Lifecycle.Started {
		panic(fmt.Sprintf("%s component must be registered during Initialize()", componentName))
	}

	redisComponent := &RedisComponent{}
	redisComponent.initialize(m.AppRuntime, componentName)
	m.AppRuntime.Components.Register(componentName, redisComponent)
	return redisComponent
}

func (m *ModuleBase) RedisLock(name ...string) *RedisLockManager {
	lockName := ""
	if len(name) > 0 {
		lockName = name[0]
	}

	componentName := redisLockManagerName(lockName)
	if component, ok := m.AppRuntime.Components.LookupComponent(componentName); ok {
		lockManager, typeOK := component.(*RedisLockManager)
		if !typeOK {
			panic(fmt.Sprintf("component %q is not a RedisLockManager", componentName))
		}
		return lockManager
	}

	if m.AppRuntime.Lifecycle.Started {
		panic(fmt.Sprintf("%s component must be registered during Initialize()", componentName))
	}

	_ = m.Redis(lockName)

	lockManager := &RedisLockManager{
		redisComponentName: redisLockRedisComponentName(lockName),
	}
	lockManager.initialize(m.AppRuntime, componentName)
	m.AppRuntime.Components.Register(componentName, lockManager)
	return lockManager
}

func (m *ModuleBase) Deduplicator(name ...string) *Deduplicator {
	dedupeName := ""
	if len(name) > 0 {
		dedupeName = name[0]
	}

	componentName := deduplicatorName(dedupeName)
	if component, ok := m.AppRuntime.Components.LookupComponent(componentName); ok {
		deduplicator, typeOK := component.(*Deduplicator)
		if !typeOK {
			panic(fmt.Sprintf("component %q is not a Deduplicator", componentName))
		}
		return deduplicator
	}

	if m.AppRuntime.Lifecycle.Started {
		panic(fmt.Sprintf("%s component must be registered during Initialize()", componentName))
	}

	_ = m.Redis(dedupeName)

	deduplicator := &Deduplicator{
		redisComponentName: deduplicatorRedisComponentName(dedupeName),
	}
	deduplicator.initialize(m.AppRuntime, componentName)
	m.AppRuntime.Components.Register(componentName, deduplicator)
	return deduplicator
}

// 在各服務呼叫Schedule 時才會lazy create
func (m *ModuleBase) Schedule() *SchedulerComponent {
	if component, ok := m.AppRuntime.Components.LookupComponent(schedulerComponentName); ok {
		scheduler, typeOK := component.(*SchedulerComponent)
		if !typeOK {
			panic("component \"Scheduler\" is not a SchedulerComponent")
		}
		return scheduler
	}

	if m.AppRuntime.Lifecycle.Started {
		panic(fmt.Sprintf("%s component must be registered during Initialize()", schedulerComponentName))
	}

	scheduler := &SchedulerComponent{}
	scheduler.initialize(m.AppRuntime, schedulerComponentName)
	m.AppRuntime.Components.Register(schedulerComponentName, scheduler)
	return scheduler
}
