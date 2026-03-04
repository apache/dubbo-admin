/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package testutils

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"dubbo-admin-ai/runtime"

	"github.com/firebase/genkit/go/genkit"
)

// SetupTestEnvironment 设置测试环境
// 返回清理函数
func SetupTestEnvironment(t *testing.T) func() {
	// 保存当前工作目录
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// 切换到项目根目录
	if err := os.Chdir("../.."); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// 返回清理函数
	return func() {
		_ = os.Chdir(origDir)
	}
}

// AssertNoError 辅助函数：断言没有错误
func AssertNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

// AssertError 辅助函数：断言有错误
func AssertError(t *testing.T, err error, msg string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected error but got nil", msg)
	}
}

// AssertEqual 辅助函数：断言相等
func AssertEqual[T comparable](t *testing.T, got, want T, msg string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: got %v, want %v", msg, got, want)
	}
}

// AssertNotEqual 辅助函数：断言不相等
func AssertNotEqual[T comparable](t *testing.T, got, notWant T, msg string) {
	t.Helper()
	if got == notWant {
		t.Fatalf("%s: got %v, should not equal %v", msg, got, notWant)
	}
}

// AssertNotNil 辅助函数：断言非空
func AssertNotNil(t *testing.T, v interface{}, msg string) {
	t.Helper()
	if v == nil {
		t.Fatalf("%s: value is nil", msg)
	}
}

// AssertNil 辅助函数：断言为空
func AssertNil(t *testing.T, v interface{}, msg string) {
	t.Helper()
	if v != nil {
		t.Fatalf("%s: expected nil but got %v", msg, v)
	}
}

// RunComponentLifecycleTest 运行组件生命周期测试
func RunComponentLifecycleTest(t *testing.T, comp runtime.Component) {
	t.Helper()

	// 测试Name
	name := comp.Name()
	if name == "" {
		t.Error("Component name should not be empty")
	}

	// 创建真实的Runtime用于Init测试
	rt := runtime.NewRuntime()
	rt.SetGenkitRegistry(CreateMockGenkitRegistry(t))

	// 测试Init
	err := comp.Init(rt)
	if err != nil {
		t.Errorf("Component.Init() failed: %v", err)
	}

	// 测试Start
	err = comp.Start()
	if err != nil {
		t.Errorf("Component.Start() failed: %v", err)
	}

	// 测试Stop
	err = comp.Stop()
	if err != nil {
		t.Errorf("Component.Stop() failed: %v", err)
	}
}

// CreateMockGenkitRegistry 创建Mock Genkit Registry
func CreateMockGenkitRegistry(t *testing.T) *genkit.Genkit {
	t.Helper()
	ctx := context.Background()
	g := genkit.Init(ctx)
	return g
}

// SetupComponentWithRuntime 使用Runtime设置组件
func SetupComponentWithRuntime(t *testing.T, comp runtime.Component) *runtime.Runtime {
	t.Helper()
	rt := runtime.NewRuntime()
	rt.SetGenkitRegistry(CreateMockGenkitRegistry(t))
	rt.RegisterComponent(comp)
	err := comp.Init(rt)
	if err != nil {
		t.Fatalf("Failed to initialize component: %v", err)
	}
	return rt
}

// CleanupComponent 清理组件资源
func CleanupComponent(t *testing.T, comp runtime.Component) {
	t.Helper()
	if err := comp.Stop(); err != nil {
		t.Errorf("Failed to stop component: %v", err)
	}
}

// GetTestAPIKey 获取测试用的API Key
func GetTestAPIKey(provider string) string {
	// 首先检查环境变量
	envKey := fmt.Sprintf("%s_API_KEY", provider)
	if key := os.Getenv(envKey); key != "" {
		return key
	}

	// 返回测试用的假Key
	return fmt.Sprintf("test-%s-api-key", provider)
}

// SkipIfMissingAPIKey 如果缺少API Key则跳过测试
func SkipIfMissingAPIKey(t *testing.T, provider string) {
	t.Helper()
	key := GetTestAPIKey(provider)
	if key == "" || key == fmt.Sprintf("test-%s-api-key", provider) {
		t.Skipf("Skipping test: %s_API_KEY not set", provider)
	}
}

// MeasureTime 测量函数执行时间
func MeasureTime(fn func()) time.Duration {
	start := time.Now()
	fn()
	return time.Since(start)
}

// AssertExecutionTime 断言执行时间在指定范围内
func AssertExecutionTime(t *testing.T, fn func(), min, max time.Duration, msg string) {
	t.Helper()
	duration := MeasureTime(fn)
	if duration < min {
		t.Errorf("%s: execution time %v is less than minimum %v", msg, duration, min)
	}
	if duration > max {
		t.Errorf("%s: execution time %v exceeds maximum %v", msg, duration, max)
	}
}
