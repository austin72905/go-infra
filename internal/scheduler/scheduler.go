package scheduler

import (
	"context"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"sync/atomic"
	"time"

	"github.com/robfig/cron/v3"
)

type JobID int

// 未來可能要加鎖同步保護
type JobInfo struct {
	ID   JobID // 管理用，方便之後remove
	Name string
	Spec string    // 頻率 cron spec："0 */5 * * * ?"
	Next time.Time // 下次預計執行時間
	Prev time.Time // 上次執行時間
}

type Engine struct {
	cron               *cron.Cron
	panicOnAnyAddError bool // 新增 job 時如果 spec 錯了或 add 失敗 要不要直接 panic
	//job 執行可能在不同 goroutine 發生，這種計數器需要 thread-safe 所以用 atomic
	runningTasks atomic.Int32 // 追蹤「目前有多少 job 還在跑」
	jobInfo      map[string]JobInfo
	stopCtx      context.Context
}

func New(location *time.Location) *Engine {
	// 不是完全用預設 parser
	// 明確指定支援哪些 cron 格式
	// 特別是 SecondOptional，表示你們允許秒欄位是可選的
	option := cron.WithParser(cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	))
	if location != nil {
		option = cron.WithLocation(location)
	}

	return &Engine{
		cron:    cron.New(option),
		jobInfo: make(map[string]JobInfo),
	}
}

// 新增 job 時如果 spec 錯了或 add 失敗 要不要直接 panic
func (e *Engine) PanicOnAnyAddError(val bool) {
	e.panicOnAnyAddError = val
}

func (e *Engine) AddFunc(spec string, process func(ctx context.Context)) (JobID, error) {
	return e.AddFuncWithName(spec, functionName(process), process)
}

func (e *Engine) AddFuncWithName(spec, action string, process func(ctx context.Context)) (JobID, error) {
	if _, exists := e.jobInfo[action]; exists {
		panic("job already exists, name=" + action)
	}

	entryID, err := e.cron.AddFunc(spec, func() {
		e.runningTasks.Add(1)
		defer e.runningTasks.Add(-1)
		process(context.Background())
	})
	if err != nil {
		if e.panicOnAnyAddError {
			panic(err)
		}
		return 0, err
	}

	info := JobInfo{
		ID:   JobID(entryID),
		Name: action,
		Spec: spec,
	}
	e.jobInfo[action] = info
	return info.ID, nil
}

func (e *Engine) Remove(id JobID) {
	e.cron.Remove(cron.EntryID(id))
}

func (e *Engine) Start() {
	e.cron.Start()
}

// 先停止新的排程觸發，並拿到一個「可用來等待所有舊 job 收完」的 context。
func (e *Engine) Stop() {
	e.stopCtx = e.cron.Stop()
}

func (e *Engine) AwaitTermination(ctx context.Context) {
	if e.stopCtx == nil {
		return
	}

	select {
	case <-ctx.Done(): // 外部傳進來的 shutdown context 超時/取消了
		return
	case <-e.stopCtx.Done(): // scheduler 裡所有已經開始跑的 job 都收完了
		return
	}
}

func (e *Engine) JobsInfo() []JobInfo {
	result := make([]JobInfo, 0, len(e.jobInfo))
	for _, info := range e.jobInfo {
		entry := e.cron.Entry(cron.EntryID(info.ID))
		info.Next = entry.Next
		info.Prev = entry.Prev
		result = append(result, info)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

func (e *Engine) RunningTasks() int {
	return int(e.runningTasks.Load())
}

func functionName(fn any) string {
	value := reflect.ValueOf(fn)
	return filepath.Base(runtime.FuncForPC(value.Pointer()).Name())
}
