package module

import "context"

// ShutdownLifecycle 是框架內部的關機生命週期協調器。
// 關機流程分成幾個具名階段，讓不同基礎設施按順序收尾。
type ShutdownLifecycle struct {
	StopIngress      []Task
	AwaitRequests    []Task
	StopBackground   []Task
	AwaitBackground  []Task
	CloseProducers   []Task
	ReleaseResources []Task
	StopServer       []Task
}

func (s *ShutdownLifecycle) AddStopIngress(task Task) {
	s.StopIngress = append(s.StopIngress, task)
}

func (s *ShutdownLifecycle) AddAwaitRequests(task Task) {
	s.AwaitRequests = append(s.AwaitRequests, task)
}

func (s *ShutdownLifecycle) AddStopBackground(task Task) {
	s.StopBackground = append(s.StopBackground, task)
}

func (s *ShutdownLifecycle) AddAwaitBackground(task Task) {
	s.AwaitBackground = append(s.AwaitBackground, task)
}

func (s *ShutdownLifecycle) AddCloseProducers(task Task) {
	s.CloseProducers = append(s.CloseProducers, task)
}

func (s *ShutdownLifecycle) AddReleaseResources(task Task) {
	s.ReleaseResources = append(s.ReleaseResources, task)
}

func (s *ShutdownLifecycle) AddStopServer(task Task) {
	s.StopServer = append(s.StopServer, task)
}

func (s *ShutdownLifecycle) run(tasks []Task, ctx context.Context) {
	for _, task := range tasks {
		task.Execute(ctx)
	}
}

func (s *ShutdownLifecycle) RunAll(ctx context.Context) {
	s.run(s.StopIngress, ctx)
	s.run(s.AwaitRequests, ctx)
	s.run(s.StopBackground, ctx)
	s.run(s.AwaitBackground, ctx)
	s.run(s.CloseProducers, ctx)
	s.run(s.ReleaseResources, ctx)
	s.run(s.StopServer, ctx)
}
