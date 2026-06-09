//go:build !cgo

package kafka

import "github.com/austin72905/go-infra/app"

func Register(runtime *app.Runtime, name string) *Component {
	stream := &Component{}
	_ = runtime
	_ = name
	return stream
}
