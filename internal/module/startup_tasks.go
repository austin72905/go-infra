package module

import "context"

// StartupLifecycle 是框架內部的啟動生命週期協調器。
// 啟動流程分成三段：
// 1. Initialize: 先建立依賴與 client
// 2. Start: 啟動背景處理器
// 3. Serve: 最後開放對外服務
type StartupLifecycle struct {
	Initialize []Task // 下面那個Task介面
	Start      []Task
	Serve      []Task
}

type Task interface {
	Execute(ctx context.Context)
}

// 轉接器
type TaskFunc func(ctx context.Context)

func (f TaskFunc) Execute(ctx context.Context) {
	f(ctx)
}

func (s *StartupLifecycle) AddInitialize(task Task) {
	s.Initialize = append(s.Initialize, task)
}

func (s *StartupLifecycle) AddStart(task Task) {
	s.Start = append(s.Start, task)
}

func (s *StartupLifecycle) AddServe(task Task) {
	s.Serve = append(s.Serve, task)
}

func (s *StartupLifecycle) RunInitialize(ctx context.Context) {
	for _, task := range s.Initialize {
		task.Execute(ctx)
	}
}

func (s *StartupLifecycle) RunStart(ctx context.Context) {
	for _, task := range s.Start {
		task.Execute(ctx)
	}
}

func (s *StartupLifecycle) RunServe(ctx context.Context) {
	for _, task := range s.Serve {
		task.Execute(ctx)
	}
}

func (s *StartupLifecycle) RunAll(ctx context.Context) {
	s.RunInitialize(ctx)
	s.RunStart(ctx)
	s.RunServe(ctx)
}
