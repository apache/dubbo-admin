package runtimetest

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	appruntime "dubbo-admin-ai/runtime"

	"gopkg.in/yaml.v3"
)

type stubComponent struct {
	name         string
	instanceName string
	calls        *[]string
	callsMu      *sync.Mutex
	validateErr  error
	initCalled   *bool
}

func (s *stubComponent) Name() string {
	if s.instanceName != "" {
		return s.instanceName
	}
	return s.name
}
func (s *stubComponent) SetName(name string) { s.instanceName = name }
func (s *stubComponent) Validate() error {
	if s.calls != nil && s.callsMu != nil {
		s.callsMu.Lock()
		*s.calls = append(*s.calls, "validate:"+s.name)
		s.callsMu.Unlock()
	}
	return s.validateErr
}
func (s *stubComponent) Init(*appruntime.Runtime) error {
	if s.calls != nil && s.callsMu != nil {
		s.callsMu.Lock()
		*s.calls = append(*s.calls, "init:"+s.name)
		s.callsMu.Unlock()
	}
	if s.initCalled != nil {
		*s.initCalled = true
	}
	return nil
}
func (s *stubComponent) Start() error { return nil }
func (s *stubComponent) Stop() error  { return nil }

func schemaDirFromRepo(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get caller")
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	return filepath.Join(root, "schema", "json")
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get caller")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

func writeRuntimeFixture(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}
}

func writeComponentFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestRuntime_RegisterFactory(t *testing.T) {
	t.Run("duplicate", func(t *testing.T) {
		rt := appruntime.NewRuntime()

		origStdout := os.Stdout
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("os.Pipe(): %v", err)
		}
		os.Stdout = w

		f1 := func(*yaml.Node) (appruntime.Component, error) { return &stubComponent{name: "c1"}, nil }
		f2 := func(*yaml.Node) (appruntime.Component, error) { return &stubComponent{name: "c2"}, nil }
		rt.RegisterFactory("dup", f1)
		rt.RegisterFactory("dup", f2)

		_ = w.Close()
		os.Stdout = origStdout
		out, _ := io.ReadAll(r)
		_ = r.Close()

		gotFactory, err := rt.GetFactoryFn("dup")
		if err != nil {
			t.Fatalf("GetFactoryFn() error: %v", err)
		}
		comp, err := gotFactory(&yaml.Node{})
		if err != nil {
			t.Fatalf("factory error: %v", err)
		}
		if comp.Name() != "c2" {
			t.Fatalf("expected second registration to overwrite, got %s", comp.Name())
		}
		if !strings.Contains(string(out), "already registered") {
			t.Fatalf("expected duplicate warning, got %q", string(out))
		}
	})

	t.Run("concurrent", func(t *testing.T) {
		rt := appruntime.NewRuntime()
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			i := i
			wg.Add(1)
			go func() {
				defer wg.Done()
				typeName := fmt.Sprintf("t-%d", i)
				rt.RegisterFactory(typeName, func(*yaml.Node) (appruntime.Component, error) {
					return &stubComponent{name: fmt.Sprintf("c-%d", i)}, nil
				})
			}()
		}
		wg.Wait()

		for i := 0; i < 100; i++ {
			if _, err := rt.GetFactoryFn(fmt.Sprintf("t-%d", i)); err != nil {
				t.Fatalf("missing registered factory t-%d: %v", i, err)
			}
		}
	})
}

func TestRuntime_Get(t *testing.T) {
	tests := []struct {
		name    string
		runFn   func(*appruntime.Runtime) error
		errLike string
	}{
		{name: "factory_not_found", runFn: func(rt *appruntime.Runtime) error { _, err := rt.GetFactoryFn("test"); return err }, errLike: "not registered"},
		{name: "component_not_found", runFn: func(rt *appruntime.Runtime) error { _, err := rt.GetComponentByType("agent"); return err }, errLike: "component type not found"},
		{name: "component_name_not_found", runFn: func(rt *appruntime.Runtime) error { _, err := rt.GetComponentByName("agent-0"); return err }, errLike: "component not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := appruntime.NewRuntime()
			err := tt.runFn(rt)
			if err == nil || !strings.Contains(err.Error(), tt.errLike) {
				t.Fatalf("expected %q error, got %v", tt.errLike, err)
			}
		})
	}
}

func TestRuntime_ComponentInitOrder(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SCHEMA_DIR", schemaDirFromRepo(t))
	writeRuntimeFixture(t, dir, "project: p\nversion: v\ncomponents:\n  logger: logger.yaml\n  memory: memory.yaml\n")
	writeComponentFile(t, dir, "logger.yaml", "type: logger\nspec: {}\n")
	writeComponentFile(t, dir, "memory.yaml", "type: memory\nspec: {}\n")

	calls := make([]string, 0)
	var mu sync.Mutex
	_, err := appruntime.Bootstrap(filepath.Join(dir, "config.yaml"), func(rt *appruntime.Runtime) {
		rt.RegisterFactory("logger", func(*yaml.Node) (appruntime.Component, error) {
			return &stubComponent{name: "logger", calls: &calls, callsMu: &mu}, nil
		})
		rt.RegisterFactory("memory", func(*yaml.Node) (appruntime.Component, error) {
			return &stubComponent{name: "memory", calls: &calls, callsMu: &mu}, nil
		})
	})
	if err != nil {
		t.Fatalf("Bootstrap() error: %v", err)
	}

	got := strings.Join(calls, ",")
	want := "validate:logger,init:logger,validate:memory,init:memory"
	if got != want {
		t.Fatalf("call order = %q, want %q", got, want)
	}
}

func TestBootstrap(t *testing.T) {
	t.Run("validate_fail_stops_init", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("SCHEMA_DIR", schemaDirFromRepo(t))
		writeRuntimeFixture(t, dir, "project: p\nversion: v\ncomponents:\n  logger: logger.yaml\n  server: server.yaml\n")
		writeComponentFile(t, dir, "logger.yaml", "type: logger\nspec: {}\n")
		writeComponentFile(t, dir, "server.yaml", "type: server\nspec: {}\n")

		serverInitCalled := false
		_, err := appruntime.Bootstrap(filepath.Join(dir, "config.yaml"), func(rt *appruntime.Runtime) {
			rt.RegisterFactory("logger", func(*yaml.Node) (appruntime.Component, error) {
				return &stubComponent{name: "logger", validateErr: fmt.Errorf("boom")}, nil
			})
			rt.RegisterFactory("server", func(*yaml.Node) (appruntime.Component, error) {
				return &stubComponent{name: "server", initCalled: &serverInitCalled}, nil
			})
		})

		if err == nil || !strings.Contains(err.Error(), "failed to validate logger") {
			t.Fatalf("expected validate fail error, got %v", err)
		}
		if serverInitCalled {
			t.Fatalf("server should not be initialized after validate failure")
		}
	})

	t.Run("missing_factory_for_configured_type", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("SCHEMA_DIR", schemaDirFromRepo(t))
		writeRuntimeFixture(t, dir, "project: p\nversion: v\ncomponents:\n  logger: logger.yaml\n")
		writeComponentFile(t, dir, "logger.yaml", "type: logger\nspec: {}\n")

		_, err := appruntime.Bootstrap(filepath.Join(dir, "config.yaml"), func(rt *appruntime.Runtime) {})
		if err == nil || !strings.Contains(err.Error(), "no factory for") {
			t.Fatalf("expected no factory error, got %v", err)
		}
	})

}

func TestRuntime_RegisterComponent(t *testing.T) {
	rt := appruntime.NewRuntime()
	comp := &stubComponent{name: "agent"}

	rt.RegisterComponent(comp)

	got, err := rt.GetComponentByName("agent-0")
	if err != nil {
		t.Fatalf("GetComponentByName(agent-0) error: %v", err)
	}
	if got != comp {
		t.Fatalf("expected stored component under explicit key")
	}
	if got.Name() != "agent-0" {
		t.Fatalf("expected injected component name agent-0, got %s", got.Name())
	}

	got, err = rt.GetComponent("agent")
	if err != nil {
		t.Fatalf("GetComponent(agent) error: %v", err)
	}
	if got != comp {
		t.Fatalf("expected component alias under Name()")
	}

	components, err := rt.GetComponentByType("agent")
	if err != nil {
		t.Fatalf("GetComponentByType(agent) error: %v", err)
	}
	if len(components) != 1 || components[0] != comp {
		t.Fatalf("expected one component lookup by type")
	}
}

func TestRuntime_RegisterComponent_MultipleOfSameType(t *testing.T) {
	rt := appruntime.NewRuntime()
	rt.RegisterComponent(&stubComponent{name: "memory"})
	rt.RegisterComponent(&stubComponent{name: "memory"})

	if _, err := rt.GetComponentByName("memory-0"); err != nil {
		t.Fatalf("expected memory-0 to be registered, got %v", err)
	}
	if _, err := rt.GetComponentByName("memory-1"); err != nil {
		t.Fatalf("expected memory-1 to be registered, got %v", err)
	}
	components, err := rt.GetComponentByType("memory")
	if err != nil {
		t.Fatalf("GetComponentByType(memory) error: %v", err)
	}
	if len(components) != 2 {
		t.Fatalf("expected two memory components, got %d", len(components))
	}
	if _, err := rt.GetComponent("memory"); err == nil || !strings.Contains(err.Error(), "multiple components found for type memory") {
		t.Fatalf("expected multiple components error, got %v", err)
	}
	got, err := rt.GetComponent("memory-1")
	if err != nil {
		t.Fatalf("GetComponent(memory-1) error: %v", err)
	}
	if got.Name() != "memory-1" {
		t.Fatalf("expected fallback lookup by name, got %s", got.Name())
	}
}

func TestRuntime_ComponentSnapshot(t *testing.T) {
	rt := appruntime.NewRuntime()
	rt.RegisterComponent(&stubComponent{name: "logger"})
	rt.RegisterComponent(&stubComponent{name: "memory"})
	rt.RegisterComponent(&stubComponent{name: "memory"})

	snapshot := appruntime.FormatComponentSnapshot(rt.ComponentSnapshot())
	if !strings.Contains(snapshot, "- logger (1): logger-0") {
		t.Fatalf("snapshot missing logger entry: %s", snapshot)
	}
	if !strings.Contains(snapshot, "- memory (2): memory-0, memory-1") {
		t.Fatalf("snapshot missing memory entry: %s", snapshot)
	}
}

func TestRuntime_GetRuntime(t *testing.T) {
	dir := t.TempDir()
	src := `package main
import "dubbo-admin-ai/runtime"
func main() { _ = runtime.GetRuntime() }`
	mainFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(mainFile, []byte(src), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	cmd := exec.Command("go", "run", mainFile)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), "GOCACHE="+filepath.Join(dir, "gocache"))
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected go run to fail with panic, got success")
	}
	if !strings.Contains(string(out), "Runtime not initialized") {
		t.Fatalf("expected panic output, got: %s", string(out))
	}
}
