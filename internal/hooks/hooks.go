package hooks

import "github.com/bird-coder/manyo/pkg/core"

func BeforeStart() error {
	return nil
}

func AfterStart() error {
	return nil
}

func BeforeStop() error {
	return nil
}

func AfterStop() error {
	core.Default().SyncLogger()
	return nil
}
