package sdk

import "testing"

func TestRunOptions(t *testing.T) {
	opt1 := WithConsole()
	opt2 := WithLab()
	var o RunOptions
	opt1(&o)
	opt2(&o)
	if !o.Console {
		t.Error("WithConsole should set Console")
	}
	if !o.Lab {
		t.Error("WithLab should set Lab")
	}
}

func TestSetRunner(t *testing.T) {
	old := runFn
	defer func() { runFn = old }()

	var called bool
	SetRunner(func(mod Exploit, opts RunOptions) {
		called = true
		if !opts.Console {
			t.Error("Console should be set")
		}
	})

	mod := &testMod{info: Info{Name: "test"}}
	Run(mod, WithConsole())
	if !called {
		t.Error("runner not called")
	}
}

func TestRunPanicWithoutRunner(t *testing.T) {
	old := runFn
	runFn = nil
	defer func() { runFn = old }()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Run should panic without runner")
		}
	}()
	Run(&testMod{})
}
