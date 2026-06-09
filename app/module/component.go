package module

// 原本叫做config,但是他的責任不止，原本命名不佳
type Component interface {
	initialize(runtime *AppRuntime, name string) // 當這個 config 第一次被 framework 建立時，怎麼把自己接到 runtime 上
	validate()                                   // 在 app 正式啟動前，檢查自己有沒有被正確設定好
}
