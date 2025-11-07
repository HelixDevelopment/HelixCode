# HelixCode Test Catalog

**Generated**: 2025-11-07 08:56:40
**Total Tests**: 1192

---

## Test Summary by Category

| Category | Count |
|----------|-------|
| E2E Test | 15 |
| Load Test | 7 |
| Other Test | 6 |
| Application Test | 40 |
| Automation Test | 24 |
| Benchmark | 13 |
| Integration Test | 23 |
| Unit Test | 1061 |
| Command Test | 3 |

---

## Application Test

### TestAddRemoveTheme

**ID**: `OT-HAR-TestAddRemoveTheme`

**Package**: `main`

**File**: `applications/harmony-os/main_test.go:205`

**Description**: No description provided

**Test Steps**:
1. Call AddTheme
2. Call GetAvailableThemes
3. Call Contains
4. Call SetTheme
5. Call Equal
   ... and 5 more steps

---

### TestAuroraAppCreation

**ID**: `OT-AUR-TestAuroraAppCreation`

**Package**: `main`

**File**: `applications/aurora-os/main_test.go:9`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call NotNil

---

### TestAuroraSecurityManager

**ID**: `OT-AUR-TestAuroraSecurityManager`

**Package**: `main`

**File**: `applications/aurora-os/main_test.go:30`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call NotNil
3. Call NotNil

---

### TestAuroraSystemMonitor

**ID**: `OT-AUR-TestAuroraSystemMonitor`

**Package**: `main`

**File**: `applications/aurora-os/main_test.go:21`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call NotNil

---

### TestCleanup

**ID**: `OT-HAR-TestCleanup`

**Package**: `main`

**File**: `applications/harmony-os/main_test.go:243`

**Description**: No description provided

**Test Steps**:
1. Call initializeHarmonyComponents
2. Call NotPanics
3. Call Cleanup
4. Call False

---

### TestCustomTheme

**ID**: `OT-HAR-TestCustomTheme`

**Package**: `main`

**File**: `applications/harmony-os/main_test.go:140`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call NotNil
3. Call Equal

---

### TestCustomTheme_Color

**ID**: `OT-DES-TestCustomTheme_Color`

**Package**: `main`

**File**: `applications/desktop/main_test.go:61`

**Description**: No description provided

**Test Steps**:
1. Call Color
2. Call NotNil

---

### TestCustomTheme_Font

**ID**: `OT-DES-TestCustomTheme_Font`

**Package**: `main`

**File**: `applications/desktop/main_test.go:77`

**Description**: No description provided

**Test Steps**:
1. Call Font
2. Call NotNil

---

### TestCustomTheme_Icon

**ID**: `OT-DES-TestCustomTheme_Icon`

**Package**: `main`

**File**: `applications/desktop/main_test.go:85`

**Description**: No description provided

**Test Steps**:
1. Call Icon
2. Call NotNil

---

### TestCustomTheme_Size

**ID**: `OT-DES-TestCustomTheme_Size`

**Package**: `main`

**File**: `applications/desktop/main_test.go:69`

**Description**: No description provided

**Test Steps**:
1. Call Size
2. Call Greater

---

### TestHarmonyDistributedEngine

**ID**: `OT-HAR-TestHarmonyDistributedEngine`

**Package**: `main`

**File**: `applications/harmony-os/main_test.go:34`

**Description**: No description provided

**Test Steps**:
1. Call initializeHarmonyComponents
2. Call NotNil
3. Call NotNil
4. Call NotNil
5. Call Equal
   ... and 4 more steps

---

### TestHarmonyIntegration

**ID**: `OT-HAR-TestHarmonyIntegration`

**Package**: `main`

**File**: `applications/harmony-os/main_test.go:19`

**Description**: No description provided

**Test Steps**:
1. Call initializeHarmonyComponents
2. Call NotNil
3. Call NotNil
4. Call NotNil
5. Call Equal
   ... and 3 more steps

---

### TestHarmonyResourceManager

**ID**: `OT-HAR-TestHarmonyResourceManager`

**Package**: `main`

**File**: `applications/harmony-os/main_test.go:74`

**Description**: No description provided

**Test Steps**:
1. Call initializeHarmonyComponents
2. Call NotNil
3. Call True
4. Call True
5. Call Equal
   ... and 2 more steps

---

### TestHarmonyServiceCoordinator

**ID**: `OT-HAR-TestHarmonyServiceCoordinator`

**Package**: `main`

**File**: `applications/harmony-os/main_test.go:90`

**Description**: No description provided

**Test Steps**:
1. Call initializeHarmonyComponents
2. Call NotNil
3. Call NotNil
4. Call NotNil
5. Call NotNil
   ... and 1 more steps

---

### TestHarmonySystemMonitor

**ID**: `OT-HAR-TestHarmonySystemMonitor`

**Package**: `main`

**File**: `applications/harmony-os/main_test.go:54`

**Description**: No description provided

**Test Steps**:
1. Call initializeHarmonyComponents
2. Call NotNil
3. Call True
4. Call Equal
5. Call updateSystemMetrics
   ... and 5 more steps

---

### TestHarmonyTheme

**ID**: `OT-HAR-TestHarmonyTheme`

**Package**: `main`

**File**: `applications/harmony-os/main_test.go:130`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call True
3. Call Equal
4. Call Equal
5. Call Equal
   ... and 2 more steps

---

### TestNewCustomTheme

**ID**: `OT-DES-TestNewCustomTheme`

**Package**: `main`

**File**: `applications/desktop/main_test.go:55`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call NotNil

---

### TestNewDesktopApp

**ID**: `OT-DES-TestNewDesktopApp`

**Package**: `main`

**File**: `applications/desktop/main_test.go:11`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call NotNil

---

### TestNewHarmonyApp

**ID**: `OT-HAR-TestNewHarmonyApp`

**Package**: `main`

**File**: `applications/harmony-os/main_test.go:11`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call NotNil
3. Call NotNil

---

### TestNewTerminalUI

**ID**: `OT-TER-TestNewTerminalUI`

**Package**: `main`

**File**: `applications/terminal-ui/main_test.go:9`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call NotNil

---

### TestNewThemeManager

**ID**: `OT-DES-TestNewThemeManager`

**Package**: `main`

**File**: `applications/desktop/main_test.go:17`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call NotNil
3. Call NotEmpty

---

### TestNewThemeManager

**ID**: `OT-TER-TestNewThemeManager`

**Package**: `main`

**File**: `applications/terminal-ui/main_test.go:15`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call NotNil
3. Call NotEmpty

---

### TestNewUIComponents

**ID**: `OT-TER-TestNewUIComponents`

**Package**: `main`

**File**: `applications/terminal-ui/main_test.go:53`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call Equal

---

### TestParseHexColor

**ID**: `OT-HAR-TestParseHexColor`

**Package**: `main`

**File**: `applications/harmony-os/main_test.go:148`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call RGBA
3. Call Equal
4. Call Equal
5. Call Equal
   ... and 1 more steps

---

### TestParseHexColor

**ID**: `OT-DES-TestParseHexColor`

**Package**: `main`

**File**: `applications/desktop/main_test.go:93`

**Description**: No description provided

**Test Steps**:
1. Call RGBA
2. Call Equal
3. Call Equal
4. Call Equal
5. Call Equal
   ... and 2 more steps

---

### TestParseHexPair

**ID**: `OT-DES-TestParseHexPair`

**Package**: `main`

**File**: `applications/desktop/main_test.go:108`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call Equal
3. Call NoError
4. Call Equal
5. Call NoError
   ... and 2 more steps

---

### TestThemeManager

**ID**: `OT-HAR-TestThemeManager`

**Package**: `main`

**File**: `applications/harmony-os/main_test.go:103`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call Equal
3. Call SetTheme
4. Call Equal
5. Call SetTheme
   ... and 5 more steps

---

### TestThemeManager_GetAvailableThemes

**ID**: `OT-TER-TestThemeManager_GetAvailableThemes`

**Package**: `main`

**File**: `applications/terminal-ui/main_test.go:22`

**Description**: No description provided

**Test Steps**:
1. Call GetAvailableThemes
2. Call Contains
3. Call Contains
4. Call Contains

---

### TestThemeManager_GetAvailableThemes

**ID**: `OT-DES-TestThemeManager_GetAvailableThemes`

**Package**: `main`

**File**: `applications/desktop/main_test.go:24`

**Description**: No description provided

**Test Steps**:
1. Call GetAvailableThemes
2. Call Contains
3. Call Contains
4. Call Contains

---

### TestThemeManager_GetColor

**ID**: `OT-TER-TestThemeManager_GetColor`

**Package**: `main`

**File**: `applications/terminal-ui/main_test.go:41`

**Description**: No description provided

**Test Steps**:
1. Call NotEmpty
2. Call GetColor
3. Call NotEmpty
4. Call GetColor
5. Call NotEmpty
   ... and 3 more steps

---

### TestThemeManager_GetColor

**ID**: `OT-DES-TestThemeManager_GetColor`

**Package**: `main`

**File**: `applications/desktop/main_test.go:43`

**Description**: No description provided

**Test Steps**:
1. Call NotEmpty
2. Call GetColor
3. Call NotEmpty
4. Call GetColor
5. Call NotEmpty
   ... and 3 more steps

---

### TestThemeManager_SetTheme

**ID**: `OT-TER-TestThemeManager_SetTheme`

**Package**: `main`

**File**: `applications/terminal-ui/main_test.go:30`

**Description**: No description provided

**Test Steps**:
1. Call True
2. Call SetTheme
3. Call Equal
4. Call GetCurrentTheme
5. Call False
   ... and 1 more steps

---

### TestThemeManager_SetTheme

**ID**: `OT-DES-TestThemeManager_SetTheme`

**Package**: `main`

**File**: `applications/desktop/main_test.go:32`

**Description**: No description provided

**Test Steps**:
1. Call True
2. Call SetTheme
3. Call Equal
4. Call GetCurrentTheme
5. Call False
   ... and 1 more steps

---

### TestUIComponents_CreateForm

**ID**: `OT-TER-TestUIComponents_CreateForm`

**Package**: `main`

**File**: `applications/terminal-ui/main_test.go:60`

**Description**: No description provided

**Test Steps**:
1. Call CreateForm
2. Call NotNil

---

### TestUIComponents_CreateList

**ID**: `OT-TER-TestUIComponents_CreateList`

**Package**: `main`

**File**: `applications/terminal-ui/main_test.go:72`

**Description**: No description provided

**Test Steps**:
1. Call CreateList
2. Call NotNil

---

### TestUIComponents_CreateLogView

**ID**: `OT-TER-TestUIComponents_CreateLogView`

**Package**: `main`

**File**: `applications/terminal-ui/main_test.go:128`

**Description**: No description provided

**Test Steps**:
1. Call CreateLogView
2. Call NotNil

---

### TestUIComponents_CreateModal

**ID**: `OT-TER-TestUIComponents_CreateModal`

**Package**: `main`

**File**: `applications/terminal-ui/main_test.go:107`

**Description**: No description provided

**Test Steps**:
1. Call CreateModal
2. Call NotNil

---

### TestUIComponents_CreateProgressBar

**ID**: `OT-TER-TestUIComponents_CreateProgressBar`

**Package**: `main`

**File**: `applications/terminal-ui/main_test.go:99`

**Description**: No description provided

**Test Steps**:
1. Call CreateProgressBar
2. Call NotNil

---

### TestUIComponents_CreateStatusBar

**ID**: `OT-TER-TestUIComponents_CreateStatusBar`

**Package**: `main`

**File**: `applications/terminal-ui/main_test.go:120`

**Description**: No description provided

**Test Steps**:
1. Call CreateStatusBar
2. Call NotNil

---

### TestUIComponents_CreateTable

**ID**: `OT-TER-TestUIComponents_CreateTable`

**Package**: `main`

**File**: `applications/terminal-ui/main_test.go:85`

**Description**: No description provided

**Test Steps**:
1. Call CreateTable
2. Call NotNil

---

## Automation Test

### TestAllFreeProvidersAutomation

**ID**: `AT-AUT-TestAllFreeProvidersAutomation`

**Package**: `automation`

**File**: `test/automation/free_providers_automation_test.go:20`

**Description**: TestAllFreeProvidersAutomation tests all free AI providers with real API calls

**Test Steps**:
1. Call Getenv
2. Call Getenv
3. Call Getenv
4. Call Getenv
5. Call Run
   ... and 5 more steps

---

### TestAnthropicProviderFullAutomation

**ID**: `AT-AUT-TestAnthropicProviderFullAutomation`

**Package**: `automation`

**File**: `test/automation/anthropic_automation_test.go:19`

**Description**: TestAnthropicProviderFullAutomation tests the Anthropic provider with real API calls

**Test Steps**:
1. Call Getenv
2. Call Skip
3. Call Getenv
4. Call Run
5. Call NewAnthropicProvider
   ... and 5 more steps

---

### TestConcurrentOperations

**ID**: `AT-AUT-TestConcurrentOperations`

**Package**: `automation`

**File**: `test/automation/automation_test.go:155`

**Description**: TestConcurrentOperations tests concurrent operations

**Test Steps**:
1. Call Short
2. Call Skip
3. Call Background
4. Call NewSSHWorkerPool
5. Call Now
   ... and 5 more steps

---

### TestDistributedTaskExecution

**ID**: `AT-AUT-TestDistributedTaskExecution`

**Package**: `automation`

**File**: `test/automation/automation_test.go:72`

**Description**: TestDistributedTaskExecution tests distributed task execution

**Test Steps**:
1. Call Short
2. Call Skip
3. Call WithTimeout
4. Call Background
5. Call NewSSHWorkerPool
   ... and 5 more steps

---

### TestErrorRecovery

**ID**: `AT-AUT-TestErrorRecovery`

**Package**: `automation`

**File**: `test/automation/automation_test.go:229`

**Description**: TestErrorRecovery tests automatic error recovery

**Test Steps**:
1. Call Short
2. Call Skip
3. Call Background
4. Call NewSSHWorkerPool
5. Call HealthCheck
   ... and 5 more steps

---

### TestFreeProvidersFeatureComparison

**ID**: `AT-AUT-TestFreeProvidersFeatureComparison`

**Package**: `automation`

**File**: `test/automation/free_providers_automation_test.go:379`

**Description**: TestFreeProvidersFeatureComparison compares features across free providers

**Test Steps**:
1. Call Getenv
2. Call Getenv
3. Call Getenv
4. Call Getenv
5. Call Logf
   ... and 5 more steps

---

### TestFreeProvidersLoadTest

**ID**: `AT-AUT-TestFreeProvidersLoadTest`

**Package**: `automation`

**File**: `test/automation/free_providers_automation_test.go:256`

**Description**: TestFreeProvidersLoadTest performs load testing on all available free providers

**Test Steps**:
1. Call Getenv
2. Call Getenv
3. Call Run
4. Call Sprintf
5. Call configFunc
   ... and 5 more steps

---

### TestGeminiProviderFullAutomation

**ID**: `AT-AUT-TestGeminiProviderFullAutomation`

**Package**: `automation`

**File**: `test/automation/gemini_automation_test.go:19`

**Description**: TestGeminiProviderFullAutomation tests the Gemini provider with real API calls

**Test Steps**:
1. Call Getenv
2. Call Getenv
3. Call Skip
4. Call Getenv
5. Call Run
   ... and 5 more steps

---

### TestLongRunningOperations

**ID**: `AT-AUT-TestLongRunningOperations`

**Package**: `automation`

**File**: `test/automation/automation_test.go:257`

**Description**: TestLongRunningOperations tests long-running operations

**Test Steps**:
1. Call Short
2. Call Skip
3. Call WithTimeout
4. Call Background
5. Call NewSSHWorkerPool
   ... and 5 more steps

---

### TestOpenRouterProviderFullAutomation

**ID**: `AT-AUT-TestOpenRouterProviderFullAutomation`

**Package**: `automation`

**File**: `test/automation/openrouter_automation_test.go:20`

**Description**: TestOpenRouterProviderFullAutomation tests the OpenRouter provider with real API calls

**Test Steps**:
1. Call Getenv
2. Call Skip
3. Call Getenv
4. Call Run
5. Call NewOpenRouterProvider
   ... and 5 more steps

---

### TestOpenRouterProviderLoadTest

**ID**: `AT-AUT-TestOpenRouterProviderLoadTest`

**Package**: `automation`

**File**: `test/automation/openrouter_automation_test.go:311`

**Description**: TestOpenRouterProviderLoadTest performs load testing with real API

**Test Steps**:
1. Call Getenv
2. Call Skip
3. Call NewOpenRouterProvider
4. Call NoError
5. Call Close
   ... and 5 more steps

---

### TestOpenRouterProviderModelCompatibility

**ID**: `AT-AUT-TestOpenRouterProviderModelCompatibility`

**Package**: `automation`

**File**: `test/automation/openrouter_automation_test.go:418`

**Description**: TestOpenRouterProviderModelCompatibility tests all available models

**Test Steps**:
1. Call Getenv
2. Call Skip
3. Call NewOpenRouterProvider
4. Call NoError
5. Call Close
   ... and 5 more steps

---

### TestOpenRouterProviderRateLimits

**ID**: `AT-AUT-TestOpenRouterProviderRateLimits`

**Package**: `automation`

**File**: `test/automation/openrouter_automation_test.go:478`

**Description**: TestOpenRouterProviderRateLimits tests rate limit handling

**Test Steps**:
1. Call Getenv
2. Call Skip
3. Call NewOpenRouterProvider
4. Call NoError
5. Call Close
   ... and 5 more steps

---

### TestPerformanceBenchmarks

**ID**: `AT-AUT-TestPerformanceBenchmarks`

**Package**: `automation`

**File**: `test/automation/automation_test.go:115`

**Description**: TestPerformanceBenchmarks tests performance benchmarks

**Test Steps**:
1. Call Short
2. Call Skip
3. Call Background
4. Call NewSSHWorkerPool
5. Call Now
   ... and 5 more steps

---

### TestQwenProviderFullAutomation

**ID**: `AT-AUT-TestQwenProviderFullAutomation`

**Package**: `automation`

**File**: `test/automation/qwen_automation_test.go:20`

**Description**: TestQwenProviderFullAutomation tests the Qwen provider with real API calls

**Test Steps**:
1. Call Getenv
2. Call Skip
3. Call Getenv
4. Call Run
5. Call NewQwenProvider
   ... and 5 more steps

---

### TestQwenProviderLoadTest

**ID**: `AT-AUT-TestQwenProviderLoadTest`

**Package**: `automation`

**File**: `test/automation/qwen_automation_test.go:313`

**Description**: TestQwenProviderLoadTest performs load testing with real API

**Test Steps**:
1. Call Getenv
2. Call Skip
3. Call NewQwenProvider
4. Call NoError
5. Call Close
   ... and 5 more steps

---

### TestQwenProviderModelCompatibility

**ID**: `AT-AUT-TestQwenProviderModelCompatibility`

**Package**: `automation`

**File**: `test/automation/qwen_automation_test.go:420`

**Description**: TestQwenProviderModelCompatibility tests all available models

**Test Steps**:
1. Call Getenv
2. Call Skip
3. Call NewQwenProvider
4. Call NoError
5. Call Close
   ... and 5 more steps

---

### TestQwenProviderRateLimits

**ID**: `AT-AUT-TestQwenProviderRateLimits`

**Package**: `automation`

**File**: `test/automation/qwen_automation_test.go:480`

**Description**: TestQwenProviderRateLimits tests rate limit handling

**Test Steps**:
1. Call Getenv
2. Call Skip
3. Call NewQwenProvider
4. Call NoError
5. Call Close
   ... and 5 more steps

---

### TestRealAIReasoning

**ID**: `AT-AUT-TestRealAIReasoning`

**Package**: `automation`

**File**: `test/automation/automation_test.go:17`

**Description**: TestRealAIReasoning tests reasoning with real AI models

**Test Steps**:
1. Call Short
2. Call Skip
3. Call WithTimeout
4. Call Background
5. Call NewProviderManager
   ... and 5 more steps

---

### TestResourceUsage

**ID**: `AT-AUT-TestResourceUsage`

**Package**: `automation`

**File**: `test/automation/automation_test.go:191`

**Description**: TestResourceUsage tests resource usage patterns

**Test Steps**:
1. Call Short
2. Call Skip
3. Call Background
4. Call NewSSHWorkerPool
5. Call NewNotificationEngine
   ... and 5 more steps

---

### TestXAIProviderFullAutomation

**ID**: `AT-AUT-TestXAIProviderFullAutomation`

**Package**: `automation`

**File**: `test/automation/xai_automation_test.go:20`

**Description**: TestXAIProviderFullAutomation tests the XAI provider with real API calls

**Test Steps**:
1. Call Getenv
2. Call Skip
3. Call Getenv
4. Call Run
5. Call NewXAIProvider
   ... and 5 more steps

---

### TestXAIProviderLoadTest

**ID**: `AT-AUT-TestXAIProviderLoadTest`

**Package**: `automation`

**File**: `test/automation/xai_automation_test.go:311`

**Description**: TestXAIProviderLoadTest performs load testing with real API

**Test Steps**:
1. Call Getenv
2. Call Skip
3. Call NewXAIProvider
4. Call NoError
5. Call Close
   ... and 5 more steps

---

### TestXAIProviderModelCompatibility

**ID**: `AT-AUT-TestXAIProviderModelCompatibility`

**Package**: `automation`

**File**: `test/automation/xai_automation_test.go:418`

**Description**: TestXAIProviderModelCompatibility tests all available models

**Test Steps**:
1. Call Getenv
2. Call Skip
3. Call NewXAIProvider
4. Call NoError
5. Call Close
   ... and 5 more steps

---

### TestXAIProviderRateLimits

**ID**: `AT-AUT-TestXAIProviderRateLimits`

**Package**: `automation`

**File**: `test/automation/xai_automation_test.go:478`

**Description**: TestXAIProviderRateLimits tests rate limit handling

**Test Steps**:
1. Call Getenv
2. Call Skip
3. Call NewXAIProvider
4. Call NoError
5. Call Close
   ... and 5 more steps

---

## Benchmark

### BenchmarkChannelThroughput

**ID**: `BM-BEN-BenchmarkChannelThroughput`

**Package**: `benchmarks`

**File**: `benchmarks/performance_bench_test.go:154`

**Description**: BenchmarkChannelThroughput measures channel throughput

**Test Steps**:
1. Call WithCancel
2. Call Background
3. Call Done
4. Call ResetTimer
5. Call ReportMetric
   ... and 2 more steps

---

### BenchmarkConcurrentMapAccess

**ID**: `BM-BEN-BenchmarkConcurrentMapAccess`

**Package**: `benchmarks`

**File**: `benchmarks/performance_bench_test.go:128`

**Description**: BenchmarkConcurrentMapAccess measures concurrent map access performance

**Test Steps**:
1. Call Sprintf
2. Call Sprintf
3. Call ResetTimer
4. Call RunParallel
5. Call Next
   ... and 5 more steps

---

### BenchmarkConcurrentWorkers

**ID**: `BM-BEN-BenchmarkConcurrentWorkers`

**Package**: `benchmarks`

**File**: `benchmarks/performance_bench_test.go:294`

**Description**: BenchmarkConcurrentWorkers simulates worker pool performance

**Test Steps**:
1. Call Add
2. Call Done
3. Call Sleep
4. Call ResetTimer
5. Call Wait
   ... and 3 more steps

---

### BenchmarkContextCancellation

**ID**: `BM-BEN-BenchmarkContextCancellation`

**Package**: `benchmarks`

**File**: `benchmarks/performance_bench_test.go:250`

**Description**: BenchmarkContextCancellation measures context cancellation overhead

**Test Steps**:
1. Call RunParallel
2. Call Next
3. Call WithCancel
4. Call Background
5. Call Done

---

### BenchmarkDatabaseMock

**ID**: `BM-BEN-BenchmarkDatabaseMock`

**Package**: `benchmarks`

**File**: `benchmarks/performance_bench_test.go:332`

**Description**: BenchmarkDatabaseMock simulates database operations

**Test Steps**:
1. Call Run
2. Call RunParallel
3. Call Next
4. Call Sprintf
5. Call Int63
   ... and 5 more steps

---

### BenchmarkGoroutineCreation

**ID**: `BM-BEN-BenchmarkGoroutineCreation`

**Package**: `benchmarks`

**File**: `benchmarks/performance_bench_test.go:179`

**Description**: BenchmarkGoroutineCreation measures goroutine creation overhead

**Test Steps**:
1. Call RunParallel
2. Call Next
3. Call ReportMetric
4. Call Seconds
5. Call Elapsed

---

### BenchmarkJSONMarshaling

**ID**: `BM-BEN-BenchmarkJSONMarshaling`

**Package**: `benchmarks`

**File**: `benchmarks/performance_bench_test.go:74`

**Description**: BenchmarkJSONMarshaling measures JSON encoding performance

**Test Steps**:
1. Call Now
2. Call Now
3. Call ResetTimer
4. Call Marshal
5. Call Fatal
   ... and 3 more steps

---

### BenchmarkJSONUnmarshaling

**ID**: `BM-BEN-BenchmarkJSONUnmarshaling`

**Package**: `benchmarks`

**File**: `benchmarks/performance_bench_test.go:101`

**Description**: BenchmarkJSONUnmarshaling measures JSON decoding performance

**Test Steps**:
1. Call Marshal
2. Call ResetTimer
3. Call Unmarshal
4. Call Fatal
5. Call ReportMetric
   ... and 2 more steps

---

### BenchmarkMemoryAllocation

**ID**: `BM-BEN-BenchmarkMemoryAllocation`

**Package**: `benchmarks`

**File**: `benchmarks/performance_bench_test.go:226`

**Description**: BenchmarkMemoryAllocation measures memory allocation patterns

**Test Steps**:
1. Call Run
2. Call ReportAllocs
3. Call Run
4. Call ReportAllocs
5. Call Run
   ... and 1 more steps

---

### BenchmarkStringConcatenation

**ID**: `BM-BEN-BenchmarkStringConcatenation`

**Package**: `benchmarks`

**File**: `benchmarks/performance_bench_test.go:194`

**Description**: BenchmarkStringConcatenation measures string concatenation performance

**Test Steps**:
1. Call Run
2. Call Run
3. Call WriteString
4. Call String
5. Call Run
   ... and 1 more steps

---

### BenchmarkTaskCreation

**ID**: `BM-BEN-BenchmarkTaskCreation`

**Package**: `benchmarks`

**File**: `benchmarks/performance_bench_test.go:16`

**Description**: BenchmarkTaskCreation measures task creation performance

**Test Steps**:
1. Call ResetTimer
2. Call RunParallel
3. Call Next
4. Call Sprintf
5. Call Int63
   ... and 5 more steps

---

### BenchmarkTaskPriorityQueue

**ID**: `BM-BEN-BenchmarkTaskPriorityQueue`

**Package**: `benchmarks`

**File**: `benchmarks/performance_bench_test.go:261`

**Description**: BenchmarkTaskPriorityQueue simulates priority queue operations

**Test Steps**:
1. Call Sprintf
2. Call Intn
3. Call ResetTimer

---

### BenchmarkTaskRetrieval

**ID**: `BM-BEN-BenchmarkTaskRetrieval`

**Package**: `benchmarks`

**File**: `benchmarks/performance_bench_test.go:44`

**Description**: BenchmarkTaskRetrieval measures task retrieval performance

**Test Steps**:
1. Call Sprintf
2. Call Sprintf
3. Call ResetTimer
4. Call RunParallel
5. Call Next
   ... and 5 more steps

---

## Command Test

### TestCLI_Run_ListModels

**ID**: `OT-CLI-TestCLI_Run_ListModels`

**Package**: `main`

**File**: `cmd/cli/main_test.go:18`

**Description**: No description provided

**Test Steps**:
1. Call NewFlagSet
2. Call Run
3. Call NoError

---

### TestNewCLI

**ID**: `OT-CLI-TestNewCLI`

**Package**: `main`

**File**: `cmd/cli/main_test.go:11`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call NotNil
3. Call NotNil

---

### TestVersionVariables

**ID**: `OT-SER-TestVersionVariables`

**Package**: `main`

**File**: `cmd/server/main_test.go:9`

**Description**: No description provided

**Test Steps**:
1. Call NotEmpty
2. Call NotEmpty
3. Call NotEmpty
4. Call Contains
5. Call Contains
   ... and 1 more steps

---

## E2E Test

### TestAnthropicProviderEndToEnd

**ID**: `E2E-E2E-TestAnthropicProviderEndToEnd`

**Package**: `e2e`

**File**: `test/e2e/anthropic_gemini_e2e_test.go:20`

**Description**: TestAnthropicProviderEndToEnd tests the Anthropic provider in a complete workflow

**Test Steps**:
1. Call Getenv
2. Call Skip
3. Call Teardown
4. Call NewAnthropicProvider
5. Call NoError
   ... and 5 more steps

---

### TestCompleteDistributedWorkflow

**ID**: `E2E-E2E-TestCompleteDistributedWorkflow`

**Package**: `e2e`

**File**: `test/e2e/comprehensive_e2e_test.go:20`

**Description**: TestCompleteDistributedWorkflow tests a complete distributed AI workflow

**Test Steps**:
1. Call Short
2. Call Skip
3. Call WithTimeout
4. Call Background
5. Call Log
   ... and 5 more steps

---

### TestCrossPlatformEndToEnd

**ID**: `E2E-E2E-TestCrossPlatformEndToEnd`

**Package**: `e2e`

**File**: `test/e2e/comprehensive_e2e_test.go:358`

**Description**: TestCrossPlatformEndToEnd tests cross-platform compatibility

**Test Steps**:
1. Call Short
2. Call Skip
3. Call Log
4. Call NewSSHWorkerPool
5. Call Background
   ... and 5 more steps

---

### TestDistributedWorkerSystem

**ID**: `E2E-E2E-TestDistributedWorkerSystem`

**Package**: `e2e`

**File**: `test/e2e/e2e_test.go:148`

**Description**: TestDistributedWorkerSystem tests the distributed worker management

**Test Steps**:
1. Call TeardownTestEnvironment
2. Call Sleep
3. Call GetAvailableWorkers
4. Call Fatal
5. Call Logf
   ... and 5 more steps

---

### TestEndToEndWorkflow

**ID**: `E2E-E2E-TestEndToEndWorkflow`

**Package**: `e2e`

**File**: `test/e2e/e2e_test.go:237`

**Description**: TestEndToEndWorkflow tests a complete workflow from task submission to completion

**Test Steps**:
1. Call TeardownTestEnvironment
2. Call Sleep
3. Call GetAvailableWorkers
4. Call Skip
5. Call SubmitTask
   ... and 5 more steps

---

### TestErrorHandling

**ID**: `E2E-E2E-TestErrorHandling`

**Package**: `e2e`

**File**: `test/e2e/e2e_test.go:286`

**Description**: TestErrorHandling tests error scenarios and recovery

**Test Steps**:
1. Call TeardownTestEnvironment
2. Call SubmitTask
3. Call Error
4. Call GetWorker
5. Call Error
   ... and 1 more steps

---

### TestFaultToleranceEndToEnd

**ID**: `E2E-E2E-TestFaultToleranceEndToEnd`

**Package**: `e2e`

**File**: `test/e2e/comprehensive_e2e_test.go:300`

**Description**: TestFaultToleranceEndToEnd tests system fault tolerance

**Test Steps**:
1. Call Short
2. Call Skip
3. Call Background
4. Call Log
5. Call NewSSHWorkerPool
   ... and 5 more steps

---

### TestGeminiProviderEndToEnd

**ID**: `E2E-E2E-TestGeminiProviderEndToEnd`

**Package**: `e2e`

**File**: `test/e2e/anthropic_gemini_e2e_test.go:314`

**Description**: TestGeminiProviderEndToEnd tests the Gemini provider in a complete workflow

**Test Steps**:
1. Call Getenv
2. Call Getenv
3. Call Skip
4. Call Teardown
5. Call NewGeminiProvider
   ... and 5 more steps

---

### TestHardwareDetection

**ID**: `E2E-E2E-TestHardwareDetection`

**Package**: `e2e`

**File**: `test/e2e/e2e_test.go:106`

**Description**: TestHardwareDetection tests the hardware detection system

**Test Steps**:
1. Call TeardownTestEnvironment
2. Call Detect
3. Call Fatalf
4. Call Error
5. Call Error
   ... and 5 more steps

---

### TestMain

**ID**: `E2E-E2E-TestMain`

**Package**: `e2e`

**File**: `test/e2e/e2e_test.go:309`

**Description**: TestMain sets up and tears down the test environment

**Test Steps**:
1. Call Println
2. Call Stat
3. Call IsNotExist
4. Call Println
5. Call Exit
   ... and 3 more steps

---

### TestModelManagement

**ID**: `E2E-E2E-TestModelManagement`

**Package**: `e2e`

**File**: `test/e2e/e2e_test.go:196`

**Description**: TestModelManagement tests the LLM model management system

**Test Steps**:
1. Call TeardownTestEnvironment
2. Call SelectOptimalModel
3. Call Logf
4. Call Error
5. Call GetAvailableModels
   ... and 4 more steps

---

### TestQwenProviderDistributedWorkflow

**ID**: `E2E-E2E-TestQwenProviderDistributedWorkflow`

**Package**: `e2e`

**File**: `test/e2e/qwen_e2e_test.go:259`

**Description**: TestQwenProviderDistributedWorkflow tests Qwen in a distributed worker scenario

**Test Steps**:
1. Call Getenv
2. Call Skip
3. Call Getenv
4. Call Skip
5. Call Teardown
   ... and 1 more steps

---

### TestQwenProviderEndToEnd

**ID**: `E2E-E2E-TestQwenProviderEndToEnd`

**Package**: `e2e`

**File**: `test/e2e/qwen_e2e_test.go:19`

**Description**: TestQwenProviderEndToEnd tests the Qwen provider in a complete workflow

**Test Steps**:
1. Call Getenv
2. Call Skip
3. Call Teardown
4. Call NewQwenProvider
5. Call NoError
   ... and 5 more steps

---

### TestRealAIEndToEnd

**ID**: `E2E-E2E-TestRealAIEndToEnd`

**Package**: `e2e`

**File**: `test/e2e/comprehensive_e2e_test.go:161`

**Description**: TestRealAIEndToEnd tests real AI integration end-to-end

**Test Steps**:
1. Call Short
2. Call Skip
3. Call WithTimeout
4. Call Background
5. Call Log
   ... and 5 more steps

---

### TestScalabilityEndToEnd

**ID**: `E2E-E2E-TestScalabilityEndToEnd`

**Package**: `e2e`

**File**: `test/e2e/comprehensive_e2e_test.go:242`

**Description**: TestScalabilityEndToEnd tests system scalability

**Test Steps**:
1. Call Short
2. Call Skip
3. Call Background
4. Call Log
5. Call NewSSHWorkerPool
   ... and 5 more steps

---

## Integration Test

### TestCrossComponentIntegration

**ID**: `IT-INT-TestCrossComponentIntegration`

**Package**: `integration`

**File**: `test/integration/integration_test.go:172`

**Description**: TestCrossComponentIntegration tests cross-component integration

**Test Steps**:
1. Call Background
2. Call NewSSHWorkerPool
3. Call NewNotificationEngine
4. Call NewMCPServer
5. Call GetWorkerStats
   ... and 5 more steps

---

### TestDiscordIntegration

**ID**: `IT-INT-TestDiscordIntegration`

**Package**: `integration`

**File**: `test/integration/discord_integration_test.go:15`

**Description**: No description provided

**Test Steps**:
1. Call NewMockDiscordServer
2. Call Close
3. Call NewDiscordChannel
4. Call Contains
5. Call Contains
   ... and 5 more steps

---

### TestDiscordIntegration_AllNotificationTypes

**ID**: `IT-INT-TestDiscordIntegration_AllNotificationTypes`

**Package**: `integration`

**File**: `test/integration/discord_integration_test.go:257`

**Description**: No description provided

**Test Steps**:
1. Call NewMockDiscordServer
2. Call Close
3. Call NewDiscordChannel
4. Call Run
5. Call Reset
   ... and 5 more steps

---

### TestDiscordIntegration_ChannelDisabled

**ID**: `IT-INT-TestDiscordIntegration_ChannelDisabled`

**Package**: `integration`

**File**: `test/integration/discord_integration_test.go:180`

**Description**: No description provided

**Test Steps**:
1. Call NewDiscordChannel
2. Call Send
3. Call Background
4. Call Error
5. Call Contains
   ... and 1 more steps

---

### TestDiscordIntegration_ConcurrentSending

**ID**: `IT-INT-TestDiscordIntegration_ConcurrentSending`

**Package**: `integration`

**File**: `test/integration/discord_integration_test.go:225`

**Description**: No description provided

**Test Steps**:
1. Call NewMockDiscordServer
2. Call Close
3. Call NewDiscordChannel
4. Call Send
5. Call Background
   ... and 3 more steps

---

### TestDiscordIntegration_MultipleNotifications

**ID**: `IT-INT-TestDiscordIntegration_MultipleNotifications`

**Package**: `integration`

**File**: `test/integration/discord_integration_test.go:159`

**Description**: No description provided

**Test Steps**:
1. Call NewMockDiscordServer
2. Call Close
3. Call NewDiscordChannel
4. Call Send
5. Call Background
   ... and 3 more steps

---

### TestDiscordIntegration_WithMetadata

**ID**: `IT-INT-TestDiscordIntegration_WithMetadata`

**Package**: `integration`

**File**: `test/integration/discord_integration_test.go:195`

**Description**: No description provided

**Test Steps**:
1. Call NewMockDiscordServer
2. Call Close
3. Call NewDiscordChannel
4. Call Send
5. Call Background
   ... and 5 more steps

---

### TestDiscordIntegration_WithNotificationEngine

**ID**: `IT-INT-TestDiscordIntegration_WithNotificationEngine`

**Package**: `integration`

**File**: `test/integration/discord_integration_test.go:113`

**Description**: No description provided

**Test Steps**:
1. Call NewMockDiscordServer
2. Call Close
3. Call NewNotificationEngine
4. Call NewDiscordChannel
5. Call RegisterChannel
   ... and 5 more steps

---

### TestDistributedWorkflow

**ID**: `IT-INT-TestDistributedWorkflow`

**Package**: `integration`

**File**: `test/integration/integration_test.go:18`

**Description**: TestDistributedWorkflow tests a complete distributed workflow

**Test Steps**:
1. Call Background
2. Call NewSSHWorkerPool
3. Call NotNil
4. Call NewNotificationEngine
5. Call NotNil
   ... and 5 more steps

---

### TestErrorHandlingIntegration

**ID**: `IT-INT-TestErrorHandlingIntegration`

**Package**: `integration`

**File**: `test/integration/integration_test.go:223`

**Description**: TestErrorHandlingIntegration tests error handling across components

**Test Steps**:
1. Call Background
2. Call NewSSHWorkerPool
3. Call NewNotificationEngine
4. Call SendDirect
5. Call NoError
   ... and 5 more steps

---

### TestLLMProviderIntegration

**ID**: `IT-INT-TestLLMProviderIntegration`

**Package**: `integration`

**File**: `test/integration/integration_test.go:71`

**Description**: TestLLMProviderIntegration tests LLM provider integration

**Test Steps**:
1. Call Background
2. Call NewProviderManager
3. Call NotNil
4. Call GetAvailableProviders
5. Call NotNil
   ... and 3 more steps

---

### TestMCPProtocolIntegration

**ID**: `IT-INT-TestMCPProtocolIntegration`

**Package**: `integration`

**File**: `test/integration/integration_test.go:135`

**Description**: TestMCPProtocolIntegration tests MCP protocol integration

**Test Steps**:
1. Call NewMCPServer
2. Call RegisterTool
3. Call NoError
4. Call Equal
5. Call GetToolCount
   ... and 3 more steps

---

### TestNotificationChannelIntegration

**ID**: `IT-INT-TestNotificationChannelIntegration`

**Package**: `integration`

**File**: `test/integration/integration_test.go:98`

**Description**: TestNotificationChannelIntegration tests notification channel integration

**Test Steps**:
1. Call Background
2. Call NewNotificationEngine
3. Call AddRule
4. Call NoError
5. Call SendNotification
   ... and 5 more steps

---

### TestSlackIntegration

**ID**: `IT-INT-TestSlackIntegration`

**Package**: `integration`

**File**: `test/integration/slack_integration_test.go:15`

**Description**: No description provided

**Test Steps**:
1. Call NewMockSlackServer
2. Call Close
3. Call NewSlackChannel
4. Call Equal
5. Call Equal
   ... and 5 more steps

---

### TestSlackIntegration_ChannelDisabled

**ID**: `IT-INT-TestSlackIntegration_ChannelDisabled`

**Package**: `integration`

**File**: `test/integration/slack_integration_test.go:196`

**Description**: No description provided

**Test Steps**:
1. Call NewSlackChannel
2. Call Send
3. Call Background
4. Call Error
5. Call Contains
   ... and 1 more steps

---

### TestSlackIntegration_MultipleNotifications

**ID**: `IT-INT-TestSlackIntegration_MultipleNotifications`

**Package**: `integration`

**File**: `test/integration/slack_integration_test.go:175`

**Description**: No description provided

**Test Steps**:
1. Call NewMockSlackServer
2. Call Close
3. Call NewSlackChannel
4. Call Send
5. Call Background
   ... and 3 more steps

---

### TestSlackIntegration_WithNotificationEngine

**ID**: `IT-INT-TestSlackIntegration_WithNotificationEngine`

**Package**: `integration`

**File**: `test/integration/slack_integration_test.go:126`

**Description**: No description provided

**Test Steps**:
1. Call NewMockSlackServer
2. Call Close
3. Call NewNotificationEngine
4. Call NewSlackChannel
5. Call RegisterChannel
   ... and 5 more steps

---

### TestTelegramIntegration

**ID**: `IT-INT-TestTelegramIntegration`

**Package**: `integration`

**File**: `test/integration/telegram_integration_test.go:16`

**Description**: No description provided

**Test Steps**:
1. Call NewMockTelegramServer
2. Call Close
3. Call Equal
4. Call Equal
5. Call Contains
   ... and 5 more steps

---

### TestTelegramIntegration_ChannelDisabled

**ID**: `IT-INT-TestTelegramIntegration_ChannelDisabled`

**Package**: `integration`

**File**: `test/integration/telegram_integration_test.go:151`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call NewTelegramChannel
3. Call False
4. Call IsEnabled
5. Call Send
   ... and 5 more steps

---

### TestTelegramIntegration_MessageFormatting

**ID**: `IT-INT-TestTelegramIntegration_MessageFormatting`

**Package**: `integration`

**File**: `test/integration/telegram_integration_test.go:139`

**Description**: No description provided

**Test Steps**:
1. Call NewTelegramChannel
2. Call True
3. Call IsEnabled
4. Call Equal
5. Call GetName

---

### TestTelegramIntegration_MultipleNotifications

**ID**: `IT-INT-TestTelegramIntegration_MultipleNotifications`

**Package**: `integration`

**File**: `test/integration/telegram_integration_test.go:235`

**Description**: No description provided

**Test Steps**:
1. Call NewTelegramChannel
2. Call Send
3. Call Background
4. Call True
5. Call IsEnabled

---

### TestTelegramIntegration_WithMockServer

**ID**: `IT-INT-TestTelegramIntegration_WithMockServer`

**Package**: `integration`

**File**: `test/integration/telegram_integration_test.go:101`

**Description**: No description provided

**Test Steps**:
1. Call NewMockTelegramServer
2. Call Close
3. Call Run
4. Call Reset
5. Call NotNil

---

### TestTelegramIntegration_WithNotificationEngine

**ID**: `IT-INT-TestTelegramIntegration_WithNotificationEngine`

**Package**: `integration`

**File**: `test/integration/telegram_integration_test.go:192`

**Description**: No description provided

**Test Steps**:
1. Call NewNotificationEngine
2. Call NewTelegramChannel
3. Call RegisterChannel
4. Call NoError
5. Call True
   ... and 5 more steps

---

## Load Test

### TestLoad_1000NotificationsPerSecond

**ID**: `LT-LOA-TestLoad_1000NotificationsPerSecond`

**Package**: `load`

**File**: `test/load/notification_load_test.go:17`

**Description**: No description provided

**Test Steps**:
1. Call Short
2. Call Skip
3. Call NewNotificationEngine
4. Call RegisterChannel
5. Call NewNotificationQueue
   ... and 5 more steps

---

### TestLoad_ConcurrentChannels

**ID**: `LT-LOA-TestLoad_ConcurrentChannels`

**Package**: `load`

**File**: `test/load/notification_load_test.go:100`

**Description**: No description provided

**Test Steps**:
1. Call Short
2. Call Skip
3. Call NewNotificationEngine
4. Call Sprintf
5. Call RegisterChannel
   ... and 5 more steps

---

### TestLoad_EventBusHighVolume

**ID**: `LT-LOA-TestLoad_EventBusHighVolume`

**Package**: `load`

**File**: `test/load/notification_load_test.go:350`

**Description**: No description provided

**Test Steps**:
1. Call Short
2. Call Skip
3. Call NewEventBus
4. Call Subscribe
5. Call AddInt64
   ... and 5 more steps

---

### TestLoad_MetricsUnderLoad

**ID**: `LT-LOA-TestLoad_MetricsUnderLoad`

**Package**: `load`

**File**: `test/load/notification_load_test.go:407`

**Description**: No description provided

**Test Steps**:
1. Call Short
2. Call Skip
3. Call NewMetrics
4. Call Now
5. Call Add
   ... and 5 more steps

---

### TestLoad_QueueSaturation

**ID**: `LT-LOA-TestLoad_QueueSaturation`

**Package**: `load`

**File**: `test/load/notification_load_test.go:169`

**Description**: No description provided

**Test Steps**:
1. Call Short
2. Call Skip
3. Call NewNotificationEngine
4. Call RegisterChannel
5. Call NewNotificationQueue
   ... and 5 more steps

---

### TestLoad_RateLimiterStress

**ID**: `LT-LOA-TestLoad_RateLimiterStress`

**Package**: `load`

**File**: `test/load/notification_load_test.go:304`

**Description**: No description provided

**Test Steps**:
1. Call Short
2. Call Skip
3. Call NewRateLimiter
4. Call NewRateLimitedChannel
5. Call NewNotificationEngine
   ... and 5 more steps

---

### TestLoad_RetryStorm

**ID**: `LT-LOA-TestLoad_RetryStorm`

**Package**: `load`

**File**: `test/load/notification_load_test.go:222`

**Description**: No description provided

**Test Steps**:
1. Call Short
2. Call Skip
3. Call AddInt64
4. Call Errorf
5. Call NewRetryableChannel
   ... and 5 more steps

---

## Other Test

### TestAuthenticationFlow

**ID**: `IT-INT-TestAuthenticationFlow`

**Package**: `integration`

**File**: `tests/integration/integration_test.go:133`

**Description**: Test Suite: Authentication Flow

**Test Steps**:
1. Call Short
2. Call Skip
3. Call Run
4. Call Post
5. Call NoError
   ... and 2 more steps

---

### TestCompleteWorkflow

**ID**: `IT-INT-TestCompleteWorkflow`

**Package**: `integration`

**File**: `tests/integration/integration_test.go:271`

**Description**: Test Suite: End-to-End Workflow

**Test Steps**:
1. Call Short
2. Call Skip
3. Call Background
4. Call Post
5. Call NoError
   ... and 5 more steps

---

### TestConcurrentTaskCreation

**ID**: `IT-INT-TestConcurrentTaskCreation`

**Package**: `integration`

**File**: `tests/integration/integration_test.go:349`

**Description**: Test Suite: Concurrent Operations

**Test Steps**:
1. Call Short
2. Call Skip
3. Call Sprintf
4. Call Post
5. Call Close
   ... and 2 more steps

---

### TestHealthEndpoint

**ID**: `IT-INT-TestHealthEndpoint`

**Package**: `integration`

**File**: `tests/integration/integration_test.go:108`

**Description**: Test Suite: Basic Health Checks

**Test Steps**:
1. Call Short
2. Call Skip
3. Call Get
4. Call NoError
5. Call Close
   ... and 5 more steps

---

### TestTaskCreationAndRetrieval

**ID**: `IT-INT-TestTaskCreationAndRetrieval`

**Package**: `integration`

**File**: `tests/integration/integration_test.go:175`

**Description**: Test Suite: Task Management

**Test Steps**:
1. Call Short
2. Call Skip
3. Call Post
4. Call NoError
5. Call Close
   ... and 5 more steps

---

### TestWorkerRegistrationAndHeartbeat

**ID**: `IT-INT-TestWorkerRegistrationAndHeartbeat`

**Package**: `integration`

**File**: `tests/integration/integration_test.go:226`

**Description**: Test Suite: Worker Management

**Test Steps**:
1. Call Short
2. Call Skip
3. Call Post
4. Call NoError
5. Call Close
   ... and 5 more steps

---

## Unit Test

### BenchmarkActionExecution

**ID**: `UT-AUT-BenchmarkActionExecution`

**Package**: `autonomy`

**File**: `internal/workflow/autonomy/autonomy_test.go:660`

**Description**: BenchmarkActionExecution benchmarks action execution

**Test Steps**:
1. Call Background
2. Call ResetTimer
3. Call Execute

---

### BenchmarkApplyCacheControl

**ID**: `UT-LLM-BenchmarkApplyCacheControl`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:1026`

**Description**: BenchmarkApplyCacheControl benchmarks cache control application

**Test Steps**:
1. Call ResetTimer

---

### BenchmarkCORSMiddleware

**ID**: `UT-SER-BenchmarkCORSMiddleware`

**Package**: `server`

**File**: `internal/server/server_test.go:229`

**Description**: No description provided

**Test Steps**:
1. Call New
2. Call Use
3. Call GET
4. Call JSON
5. Call ResetTimer
   ... and 3 more steps

---

### BenchmarkCacheGetHit

**ID**: `UT-WEB-BenchmarkCacheGetHit`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:636`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Set
3. Call ResetTimer
4. Call Get

---

### BenchmarkCacheMetricsUpdate

**ID**: `UT-LLM-BenchmarkCacheMetricsUpdate`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:1063`

**Description**: BenchmarkCacheMetricsUpdate benchmarks metrics update

**Test Steps**:
1. Call ResetTimer
2. Call UpdateMetrics

---

### BenchmarkCalculateCacheSavings

**ID**: `UT-LLM-BenchmarkCalculateCacheSavings`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:1048`

**Description**: BenchmarkCalculateCacheSavings benchmarks savings calculation

**Test Steps**:
1. Call ResetTimer

---

### BenchmarkChecksumCalculation

**ID**: `UT-MAP-BenchmarkChecksumCalculation`

**Package**: `mapping`

**File**: `internal/tools/mapping/mapping_test.go:794`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer

---

### BenchmarkComparison_SyncVsAsync_Async

**ID**: `UT-EVE-BenchmarkComparison_SyncVsAsync_Async`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:602`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call ResetTimer
3. Call Publish
4. Call Background

---

### BenchmarkComparison_SyncVsAsync_Sync

**ID**: `UT-EVE-BenchmarkComparison_SyncVsAsync_Sync`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:582`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call ResetTimer
3. Call Publish
4. Call Background

---

### BenchmarkCompleteStack_WithQueueAndMetrics

**ID**: `UT-NOT-BenchmarkCompleteStack_WithQueueAndMetrics`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:480`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call Start
3. Call Stop
4. Call ResetTimer
5. Call Now
   ... and 5 more steps

---

### BenchmarkCompleteStack_WithRetryAndRateLimit

**ID**: `UT-NOT-BenchmarkCompleteStack_WithRetryAndRateLimit`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:451`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call ResetTimer
3. Call SendDirect
4. Call Background

---

### BenchmarkCompressionCoordinator_Compress

**ID**: `UT-COM-BenchmarkCompressionCoordinator_Compress`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:651`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call Compress
3. Call Background

---

### BenchmarkConcurrentExecution

**ID**: `UT-SHE-BenchmarkConcurrentExecution`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:567`

**Description**: BenchmarkConcurrentExecution benchmarks concurrent execution

**Test Steps**:
1. Call ResetTimer
2. Call RunParallel
3. Call Next
4. Call Sprintf
5. Call Execute
   ... and 1 more steps

---

### BenchmarkDiffAnalysis

**ID**: `UT-GIT-BenchmarkDiffAnalysis`

**Package**: `git`

**File**: `internal/tools/git/git_test.go:882`

**Description**: Benchmark tests

**Test Steps**:
1. Call ResetTimer
2. Call Analyze
3. Call Background

---

### BenchmarkDiscordChannel_Send

**ID**: `UT-NOT-BenchmarkDiscordChannel_Send`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:27`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call Send
3. Call Background

---

### BenchmarkEngine_1000Channels

**ID**: `UT-NOT-BenchmarkEngine_1000Channels`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:402`

**Description**: No description provided

**Test Steps**:
1. Call Sprintf
2. Call RegisterChannel
3. Call ResetTimer
4. Call SendDirect
5. Call Background

---

### BenchmarkEngine_LargePayload

**ID**: `UT-NOT-BenchmarkEngine_LargePayload`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:597`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call Sprintf
3. Call Sprintf
4. Call ResetTimer
5. Call SendDirect
   ... and 1 more steps

---

### BenchmarkEngine_MediumPayload

**ID**: `UT-NOT-BenchmarkEngine_MediumPayload`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:575`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call ResetTimer
3. Call SendDirect
4. Call Background

---

### BenchmarkEngine_RuleEvaluation_10Rules

**ID**: `UT-NOT-BenchmarkEngine_RuleEvaluation_10Rules`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:645`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call AddRule
3. Call Sprintf
4. Call Sprintf
5. Call ResetTimer
   ... and 2 more steps

---

### BenchmarkEngine_RuleEvaluation_NoRules

**ID**: `UT-NOT-BenchmarkEngine_RuleEvaluation_NoRules`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:628`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call ResetTimer
3. Call SendNotification
4. Call Background

---

### BenchmarkEngine_SendDirect_Allocations

**ID**: `UT-NOT-BenchmarkEngine_SendDirect_Allocations`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:381`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call ReportAllocs
3. Call ResetTimer
4. Call SendDirect
5. Call Background

---

### BenchmarkEngine_SmallPayload

**ID**: `UT-NOT-BenchmarkEngine_SmallPayload`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:558`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call ResetTimer
3. Call SendDirect
4. Call Background

---

### BenchmarkEventBus_AllTaskEventTypes

**ID**: `UT-EVE-BenchmarkEventBus_AllTaskEventTypes`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:499`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call ResetTimer
3. Call Publish
4. Call Background

---

### BenchmarkEventBus_ConcurrentMixed

**ID**: `UT-EVE-BenchmarkEventBus_ConcurrentMixed`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:158`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call RunParallel
3. Call Next
4. Call Subscribe
5. Call Publish
   ... and 1 more steps

---

### BenchmarkEventBus_ConcurrentPublish

**ID**: `UT-EVE-BenchmarkEventBus_ConcurrentPublish`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:122`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call ResetTimer
3. Call RunParallel
4. Call Next
5. Call Publish
   ... and 1 more steps

---

### BenchmarkEventBus_ConcurrentSubscribe

**ID**: `UT-EVE-BenchmarkEventBus_ConcurrentSubscribe`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:144`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call RunParallel
3. Call Next
4. Call Subscribe

---

### BenchmarkEventBus_GetErrors

**ID**: `UT-EVE-BenchmarkEventBus_GetErrors`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:555`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call Errorf
3. Call Publish
4. Call Background
5. Call ResetTimer
   ... and 1 more steps

---

### BenchmarkEventBus_LargePayload

**ID**: `UT-EVE-BenchmarkEventBus_LargePayload`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:327`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call Sprintf
3. Call Sprintf
4. Call ResetTimer
5. Call Publish
   ... and 1 more steps

---

### BenchmarkEventBus_MediumPayload

**ID**: `UT-EVE-BenchmarkEventBus_MediumPayload`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:300`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call ResetTimer
3. Call Publish
4. Call Background

---

### BenchmarkEventBus_PublishAllocations

**ID**: `UT-EVE-BenchmarkEventBus_PublishAllocations`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:394`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call ReportAllocs
3. Call ResetTimer
4. Call Publish
5. Call Background

---

### BenchmarkEventBus_PublishAsync

**ID**: `UT-EVE-BenchmarkEventBus_PublishAsync`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:61`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call ResetTimer
3. Call Publish
4. Call Background

---

### BenchmarkEventBus_PublishMultipleSubscribers

**ID**: `UT-EVE-BenchmarkEventBus_PublishMultipleSubscribers`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:97`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call ResetTimer
3. Call Publish
4. Call Background

---

### BenchmarkEventBus_PublishNoSubscribers

**ID**: `UT-EVE-BenchmarkEventBus_PublishNoSubscribers`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:81`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call Publish
3. Call Background

---

### BenchmarkEventBus_PublishSync

**ID**: `UT-EVE-BenchmarkEventBus_PublishSync`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:41`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call ResetTimer
3. Call Publish
4. Call Background

---

### BenchmarkEventBus_PublishWithErrors

**ID**: `UT-EVE-BenchmarkEventBus_PublishWithErrors`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:535`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call Errorf
3. Call ResetTimer
4. Call Publish
5. Call Background

---

### BenchmarkEventBus_SmallPayload

**ID**: `UT-EVE-BenchmarkEventBus_SmallPayload`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:281`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call ResetTimer
3. Call Publish
4. Call Background

---

### BenchmarkEventBus_Subscribe

**ID**: `UT-EVE-BenchmarkEventBus_Subscribe`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:12`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call Subscribe

---

### BenchmarkEventBus_SystemEvents

**ID**: `UT-EVE-BenchmarkEventBus_SystemEvents`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:256`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call ResetTimer
3. Call Publish
4. Call Background

---

### BenchmarkEventBus_TaskEvents

**ID**: `UT-EVE-BenchmarkEventBus_TaskEvents`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:184`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call ResetTimer
3. Call Publish
4. Call Background

---

### BenchmarkEventBus_Unsubscribe

**ID**: `UT-EVE-BenchmarkEventBus_Unsubscribe`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:24`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call ResetTimer
3. Call Unsubscribe

---

### BenchmarkEventBus_WorkerEvents

**ID**: `UT-EVE-BenchmarkEventBus_WorkerEvents`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:232`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call ResetTimer
3. Call Publish
4. Call Background

---

### BenchmarkEventBus_WorkflowEvents

**ID**: `UT-EVE-BenchmarkEventBus_WorkflowEvents`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:208`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call ResetTimer
3. Call Publish
4. Call Background

---

### BenchmarkEvent_Creation

**ID**: `UT-EVE-BenchmarkEvent_Creation`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:377`

**Description**: No description provided

**Test Steps**:
1. Call ReportAllocs
2. Call ResetTimer

---

### BenchmarkFetch

**ID**: `UT-WEB-BenchmarkFetch`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:597`

**Description**: Benchmark tests

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Write
4. Call Close
5. Call Close
   ... and 3 more steps

---

### BenchmarkGlobalBus_Publish

**ID**: `UT-EVE-BenchmarkGlobalBus_Publish`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:355`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call ResetTimer
3. Call Publish
4. Call Background

---

### BenchmarkImageDetection

**ID**: `UT-VIS-BenchmarkImageDetection`

**Package**: `vision`

**File**: `internal/llm/vision/vision_test.go:666`

**Description**: BenchmarkImageDetection benchmarks image detection performance

**Test Steps**:
1. Call Background
2. Call Repeat
3. Call ResetTimer
4. Call Detect
5. Call Fatal

---

### BenchmarkLanguageDetection

**ID**: `UT-MAP-BenchmarkLanguageDetection`

**Package**: `mapping`

**File**: `internal/tools/mapping/mapping_test.go:803`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer

---

### BenchmarkMessageCache

**ID**: `UT-GIT-BenchmarkMessageCache`

**Package**: `git`

**File**: `internal/tools/git/git_test.go:906`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call Set
3. Call Get

---

### BenchmarkMetrics_ConcurrentWrites

**ID**: `UT-NOT-BenchmarkMetrics_ConcurrentWrites`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:322`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call RunParallel
3. Call Next
4. Call RecordSent

---

### BenchmarkMetrics_GetMetrics

**ID**: `UT-NOT-BenchmarkMetrics_GetMetrics`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:279`

**Description**: No description provided

**Test Steps**:
1. Call RecordSent
2. Call RecordFailed
3. Call ResetTimer
4. Call GetMetrics

---

### BenchmarkMetrics_GetSuccessRate

**ID**: `UT-NOT-BenchmarkMetrics_GetSuccessRate`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:290`

**Description**: No description provided

**Test Steps**:
1. Call RecordSent
2. Call RecordFailed
3. Call ResetTimer
4. Call GetSuccessRate

---

### BenchmarkMetrics_RecordFailed

**ID**: `UT-NOT-BenchmarkMetrics_RecordFailed`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:270`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call RecordFailed

---

### BenchmarkMetrics_RecordSent

**ID**: `UT-NOT-BenchmarkMetrics_RecordSent`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:261`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call RecordSent

---

### BenchmarkNewServer

**ID**: `UT-SER-BenchmarkNewServer`

**Package**: `server`

**File**: `internal/server/server_test.go:219`

**Description**: Benchmark tests

**Test Steps**:
1. Call ResetTimer

---

### BenchmarkNotificationEngine_ConcurrentSends

**ID**: `UT-NOT-BenchmarkNotificationEngine_ConcurrentSends`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:303`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call ResetTimer
3. Call RunParallel
4. Call Next
5. Call SendDirect
   ... and 1 more steps

---

### BenchmarkNotificationEngine_RegisterChannel

**ID**: `UT-NOT-BenchmarkNotificationEngine_RegisterChannel`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:57`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call Sprintf
3. Call RegisterChannel

---

### BenchmarkNotificationEngine_SendDirect

**ID**: `UT-NOT-BenchmarkNotificationEngine_SendDirect`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:67`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call ResetTimer
3. Call SendDirect
4. Call Background

---

### BenchmarkNotificationEngine_SendNotificationWithRules

**ID**: `UT-NOT-BenchmarkNotificationEngine_SendNotificationWithRules`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:84`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call AddRule
3. Call ResetTimer
4. Call SendNotification
5. Call Background

---

### BenchmarkNotificationQueue_ConcurrentEnqueue

**ID**: `UT-NOT-BenchmarkNotificationQueue_ConcurrentEnqueue`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:344`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call RunParallel
3. Call Next
4. Call Enqueue

---

### BenchmarkNotificationQueue_Dequeue

**ID**: `UT-NOT-BenchmarkNotificationQueue_Dequeue`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:212`

**Description**: No description provided

**Test Steps**:
1. Call Enqueue
2. Call ResetTimer
3. Call Dequeue

---

### BenchmarkNotificationQueue_Enqueue

**ID**: `UT-NOT-BenchmarkNotificationQueue_Enqueue`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:196`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call Enqueue

---

### BenchmarkNotificationQueue_Throughput

**ID**: `UT-NOT-BenchmarkNotificationQueue_Throughput`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:233`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call Start
3. Call Stop
4. Call ResetTimer
5. Call Enqueue
   ... and 2 more steps

---

### BenchmarkNotification_Creation

**ID**: `UT-NOT-BenchmarkNotification_Creation`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:364`

**Description**: No description provided

**Test Steps**:
1. Call ReportAllocs
2. Call ResetTimer

---

### BenchmarkOptionRanking

**ID**: `UT-PLA-BenchmarkOptionRanking`

**Package**: `planmode`

**File**: `internal/workflow/planmode/planmode_test.go:768`

**Description**: No description provided

**Test Steps**:
1. Call String
2. Call New
3. Call Duration
4. Call NewReader
5. Call ResetTimer
   ... and 1 more steps

---

### BenchmarkParse

**ID**: `UT-WEB-BenchmarkParse`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:615`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call Parse

---

### BenchmarkPermissionCheck

**ID**: `UT-AUT-BenchmarkPermissionCheck`

**Package**: `autonomy`

**File**: `internal/workflow/autonomy/autonomy_test.go:646`

**Description**: BenchmarkPermissionCheck benchmarks permission checking

**Test Steps**:
1. Call Background
2. Call ResetTimer
3. Call Check

---

### BenchmarkPlanGeneration

**ID**: `UT-PLA-BenchmarkPlanGeneration`

**Package**: `planmode`

**File**: `internal/workflow/planmode/planmode_test.go:753`

**Description**: Benchmark tests

**Test Steps**:
1. Call ResetTimer
2. Call GeneratePlan
3. Call Background

---

### BenchmarkQueue_HighVolume_10Workers

**ID**: `UT-NOT-BenchmarkQueue_HighVolume_10Workers`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:423`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call Start
3. Call Stop
4. Call ResetTimer
5. Call Enqueue
   ... and 2 more steps

---

### BenchmarkQueue_Parallel_10Workers

**ID**: `UT-NOT-BenchmarkQueue_Parallel_10Workers`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:520`

**Description**: No description provided

**Test Steps**:
1. Execute test logic

---

### BenchmarkQueue_Parallel_1Worker

**ID**: `UT-NOT-BenchmarkQueue_Parallel_1Worker`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:512`

**Description**: No description provided

**Test Steps**:
1. Execute test logic

---

### BenchmarkQueue_Parallel_20Workers

**ID**: `UT-NOT-BenchmarkQueue_Parallel_20Workers`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:524`

**Description**: No description provided

**Test Steps**:
1. Execute test logic

---

### BenchmarkQueue_Parallel_5Workers

**ID**: `UT-NOT-BenchmarkQueue_Parallel_5Workers`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:516`

**Description**: No description provided

**Test Steps**:
1. Execute test logic

---

### BenchmarkRateLimitedChannel_Send

**ID**: `UT-NOT-BenchmarkRateLimitedChannel_Send`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:177`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call Send
3. Call Background

---

### BenchmarkRateLimiter_Allow

**ID**: `UT-NOT-BenchmarkRateLimiter_Allow`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:168`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call Allow

---

### BenchmarkRateLimiter_ConcurrentAllow

**ID**: `UT-NOT-BenchmarkRateLimiter_ConcurrentAllow`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:333`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call RunParallel
3. Call Next
4. Call Allow

---

### BenchmarkRetryableChannel_SendSuccess

**ID**: `UT-NOT-BenchmarkRetryableChannel_SendSuccess`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:110`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call Send
3. Call Background

---

### BenchmarkRetryableChannel_SendWithRetries

**ID**: `UT-NOT-BenchmarkRetryableChannel_SendWithRetries`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:132`

**Description**: No description provided

**Test Steps**:
1. Call Errorf
2. Call ResetTimer
3. Call Send
4. Call Background

---

### BenchmarkSecurityMiddleware

**ID**: `UT-SER-BenchmarkSecurityMiddleware`

**Package**: `server`

**File**: `internal/server/server_test.go:246`

**Description**: No description provided

**Test Steps**:
1. Call New
2. Call Use
3. Call GET
4. Call JSON
5. Call ResetTimer
   ... and 3 more steps

---

### BenchmarkSimpleExecution

**ID**: `UT-SHE-BenchmarkSimpleExecution`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:553`

**Description**: BenchmarkSimpleExecution benchmarks simple command execution

**Test Steps**:
1. Call ResetTimer
2. Call Sprintf
3. Call Execute
4. Call Background

---

### BenchmarkSlackChannel_Send

**ID**: `UT-NOT-BenchmarkSlackChannel_Send`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:13`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call Send
3. Call Background

---

### BenchmarkSlidingWindowStrategy

**ID**: `UT-COM-BenchmarkSlidingWindowStrategy`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:640`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call Execute
3. Call Background

---

### BenchmarkStress_100ConcurrentSends

**ID**: `UT-NOT-BenchmarkStress_100ConcurrentSends`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:674`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call ResetTimer
3. Call Add
4. Call Done
5. Call SendDirect
   ... and 2 more steps

---

### BenchmarkStress_ConcurrentSubscribePublish

**ID**: `UT-EVE-BenchmarkStress_ConcurrentSubscribePublish`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:463`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call Add
3. Call Done
4. Call Subscribe
5. Call Add
   ... and 4 more steps

---

### BenchmarkStress_HighVolumePublish

**ID**: `UT-EVE-BenchmarkStress_HighVolumePublish`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:418`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call ResetTimer
3. Call Publish
4. Call Background

---

### BenchmarkStress_ManySubscribers

**ID**: `UT-EVE-BenchmarkStress_ManySubscribers`

**Package**: `event`

**File**: `internal/event/benchmark_test.go:440`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call ResetTimer
3. Call Publish
4. Call Background

---

### BenchmarkStress_RateLimiterContention

**ID**: `UT-NOT-BenchmarkStress_RateLimiterContention`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:699`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call Add
3. Call Done
4. Call Allow
5. Call Wait

---

### BenchmarkTelegramChannel_Send

**ID**: `UT-NOT-BenchmarkTelegramChannel_Send`

**Package**: `notification`

**File**: `internal/notification/benchmark_test.go:41`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call Send
3. Call Background

---

### BenchmarkTokenCounter_Count

**ID**: `UT-COM-BenchmarkTokenCounter_Count`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:630`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call Count

---

### BenchmarkTokenCounting

**ID**: `UT-MAP-BenchmarkTokenCounting`

**Package**: `mapping`

**File**: `internal/tools/mapping/mapping_test.go:784`

**Description**: No description provided

**Test Steps**:
1. Call ResetTimer
2. Call Count

---

### BenchmarkToolExecution

**ID**: `UT-TOO-BenchmarkToolExecution`

**Package**: `tools`

**File**: `internal/tools/registry_test.go:361`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call Fatalf
4. Call Close
5. Call Background
   ... and 5 more steps

---

### TestActionExecution

**ID**: `UT-AUT-TestActionExecution`

**Package**: `autonomy`

**File**: `internal/workflow/autonomy/autonomy_test.go:394`

**Description**: TestActionExecution tests action execution

**Test Steps**:
1. Call Background
2. Call Execute
3. Call Fatalf
4. Call Error
5. Call Error

---

### TestAddRule

**ID**: `UT-NOT-TestAddRule`

**Package**: `notification`

**File**: `internal/notification/engine_test.go:32`

**Description**: No description provided

**Test Steps**:
1. Call AddRule
2. Call NoError
3. Call Equal
4. Call NotEqual

---

### TestAgentRegistry

**ID**: `UT-AGE-TestAgentRegistry`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:378`

**Description**: No description provided

**Test Steps**:
1. Call Count
2. Call Errorf
3. Call Count
4. Call Register
5. Call Errorf
   ... and 5 more steps

---

### TestAgentRegistryByCapability

**ID**: `UT-AGE-TestAgentRegistryByCapability`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:458`

**Description**: No description provided

**Test Steps**:
1. Call Register
2. Call Register
3. Call GetByCapability
4. Call Errorf
5. Call GetByCapability
   ... and 3 more steps

---

### TestAgentRegistryGetByCapabilityEmpty

**ID**: `UT-AGE-TestAgentRegistryGetByCapabilityEmpty`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:599`

**Description**: No description provided

**Test Steps**:
1. Call GetByCapability
2. Call Errorf
3. Call Register
4. Call GetByCapability
5. Call Errorf

---

### TestAgentRegistryGetByTypeEmpty

**ID**: `UT-AGE-TestAgentRegistryGetByTypeEmpty`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:573`

**Description**: No description provided

**Test Steps**:
1. Call GetByType
2. Call Errorf
3. Call Register
4. Call GetByType
5. Call Errorf

---

### TestAgentRegistryList

**ID**: `UT-AGE-TestAgentRegistryList`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:476`

**Description**: TestAgentRegistryList tests the List method

**Test Steps**:
1. Call Register
2. Call Register
3. Call Register
4. Call List
5. Call Len
   ... and 4 more steps

---

### TestAgentRegistryListMultiple

**ID**: `UT-AGE-TestAgentRegistryListMultiple`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:505`

**Description**: No description provided

**Test Steps**:
1. Call List
2. Call Errorf
3. Call Register
4. Call Register
5. Call List
   ... and 1 more steps

---

### TestAgentRegistryMultipleUnregister

**ID**: `UT-AGE-TestAgentRegistryMultipleUnregister`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:550`

**Description**: No description provided

**Test Steps**:
1. Call Register
2. Call Unregister
3. Call Unregister
4. Call Unregister
5. Call Count
   ... and 2 more steps

---

### TestAgentRegistryUnregisterNonExistent

**ID**: `UT-AGE-TestAgentRegistryUnregisterNonExistent`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:539`

**Description**: No description provided

**Test Steps**:
1. Call Unregister
2. Call Count
3. Call Errorf
4. Call Count

---

### TestAllNotificationChannels

**ID**: `UT-NOT-TestAllNotificationChannels`

**Package**: `notification`

**File**: `internal/notification/engine_test.go:141`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call RegisterChannel
3. Call NoError
4. Call RegisterChannel
5. Call NoError
   ... and 5 more steps

---

### TestAllowlistEnforcement

**ID**: `UT-SHE-TestAllowlistEnforcement`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:128`

**Description**: TestAllowlistEnforcement tests allowlist enforcement

**Test Steps**:
1. Call Run
2. Call Execute
3. Call Background
4. Call NoError
5. Call Equal
   ... and 5 more steps

---

### TestAmendDetector

**ID**: `UT-GIT-TestAmendDetector`

**Package**: `git`

**File**: `internal/tools/git/git_test.go:334`

**Description**: TestAmendDetector tests amend detection

**Test Steps**:
1. Call Run
2. Call Command
3. Call Run
4. Call Fatal
5. Call Join
   ... and 5 more steps

---

### TestAnalyzePolicy

**ID**: `UT-COM-TestAnalyzePolicy`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:492`

**Description**: Test 18: Analyze Policy

**Test Steps**:
1. Call Equal
2. Call Greater
3. Call NotEmpty
4. Call GreaterOrEqual
5. Call LessOrEqual

---

### TestAnthropicProvider_Close

**ID**: `UT-LLM-TestAnthropicProvider_Close`

**Package**: `llm`

**File**: `internal/llm/anthropic_provider_test.go:601`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call Close
3. Call NoError

---

### TestAnthropicProvider_ErrorHandling

**ID**: `UT-LLM-TestAnthropicProvider_ErrorHandling`

**Package**: `llm`

**File**: `internal/llm/anthropic_provider_test.go:487`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call NewServer
3. Call HandlerFunc
4. Call WriteHeader
5. Call Write
   ... and 5 more steps

---

### TestAnthropicProvider_ExtendedThinking

**ID**: `UT-LLM-TestAnthropicProvider_ExtendedThinking`

**Package**: `llm`

**File**: `internal/llm/anthropic_provider_test.go:354`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Decode
4. Call NewDecoder
5. Call NoError
   ... and 5 more steps

---

### TestAnthropicProvider_Generate

**ID**: `UT-LLM-TestAnthropicProvider_Generate`

**Package**: `llm`

**File**: `internal/llm/anthropic_provider_test.go:191`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Equal
4. Call Get
5. Call Equal
   ... and 5 more steps

---

### TestAnthropicProvider_GenerateWithTools

**ID**: `UT-LLM-TestAnthropicProvider_GenerateWithTools`

**Package**: `llm`

**File**: `internal/llm/anthropic_provider_test.go:260`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Decode
4. Call NewDecoder
5. Call NoError
   ... and 5 more steps

---

### TestAnthropicProvider_GetCapabilities

**ID**: `UT-LLM-TestAnthropicProvider_GetCapabilities`

**Package**: `llm`

**File**: `internal/llm/anthropic_provider_test.go:139`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call GetCapabilities
3. Call NotEmpty
4. Call True
5. Call True
   ... and 3 more steps

---

### TestAnthropicProvider_GetHealth

**ID**: `UT-LLM-TestAnthropicProvider_GetHealth`

**Package**: `llm`

**File**: `internal/llm/anthropic_provider_test.go:564`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Encode
4. Call NewEncoder
5. Call Close
   ... and 5 more steps

---

### TestAnthropicProvider_GetModels

**ID**: `UT-LLM-TestAnthropicProvider_GetModels`

**Package**: `llm`

**File**: `internal/llm/anthropic_provider_test.go:112`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call GetModels
3. Call NotEmpty
4. Call Equal
5. Call Greater
   ... and 5 more steps

---

### TestAnthropicProvider_GetName

**ID**: `UT-LLM-TestAnthropicProvider_GetName`

**Package**: `llm`

**File**: `internal/llm/anthropic_provider_test.go:101`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call Equal
3. Call GetName

---

### TestAnthropicProvider_GetType

**ID**: `UT-LLM-TestAnthropicProvider_GetType`

**Package**: `llm`

**File**: `internal/llm/anthropic_provider_test.go:90`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call Equal
3. Call GetType

---

### TestAnthropicProvider_IsAvailable

**ID**: `UT-LLM-TestAnthropicProvider_IsAvailable`

**Package**: `llm`

**File**: `internal/llm/anthropic_provider_test.go:164`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call NoError
3. Call IsAvailable
4. Call Background
5. Call Equal

---

### TestAnthropicProvider_PromptCaching

**ID**: `UT-LLM-TestAnthropicProvider_PromptCaching`

**Package**: `llm`

**File**: `internal/llm/anthropic_provider_test.go:414`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Decode
4. Call NewDecoder
5. Call NoError
   ... and 5 more steps

---

### TestApplyReasoningBudget_ExceedsBudget

**ID**: `UT-LLM-TestApplyReasoningBudget_ExceedsBudget`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:211`

**Description**: No description provided

**Test Steps**:
1. Call Repeat
2. Call NotEqual
3. Call LessOrEqual
4. Call Less

---

### TestApplyReasoningBudget_UnlimitedBudget

**ID**: `UT-LLM-TestApplyReasoningBudget_UnlimitedBudget`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:173`

**Description**: Test ApplyReasoningBudget

**Test Steps**:
1. Call Equal
2. Call Equal

---

### TestApplyReasoningBudget_WithinBudget

**ID**: `UT-LLM-TestApplyReasoningBudget_WithinBudget`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:192`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call Equal

---

### TestAreDependenciesCompleted

**ID**: `UT-WOR-TestAreDependenciesCompleted`

**Package**: `workflow`

**File**: `internal/workflow/executor_test.go:178`

**Description**: No description provided

**Test Steps**:
1. Call NewManager
2. Call True
3. Call areDependenciesCompleted
4. Call False
5. Call areDependenciesCompleted

---

### TestArtifactTypes

**ID**: `UT-TAS-TestArtifactTypes`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:536`

**Description**: No description provided

**Test Steps**:
1. Call Now
2. Call Equal
3. Call NotEmpty
4. Call NotEmpty
5. Call NotEmpty
   ... and 3 more steps

---

### TestAttributionManager

**ID**: `UT-GIT-TestAttributionManager`

**Package**: `git`

**File**: `internal/tools/git/git_test.go:415`

**Description**: TestAttributionManager tests attribution management

**Test Steps**:
1. Call Run
2. Call AddAttribution
3. Call Contains
4. Call Errorf
5. Call Run
   ... and 5 more steps

---

### TestAttributionParsing

**ID**: `UT-GIT-TestAttributionParsing`

**Package**: `git`

**File**: `internal/tools/git/git_test.go:813`

**Description**: TestAttributionParsing tests attribution parsing

**Test Steps**:
1. Call Run
2. Call Fatalf
3. Call Errorf
4. Call Errorf
5. Call Errorf
   ... and 4 more steps

---

### TestAudioRecorder_DoubleStart

**ID**: `UT-VOI-TestAudioRecorder_DoubleStart`

**Package**: `voice`

**File**: `internal/tools/voice/voice_test.go:166`

**Description**: TestAudioRecorder_DoubleStart tests error handling for double start

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Fatalf
5. Call Background
   ... and 5 more steps

---

### TestAudioRecorder_StartStop

**ID**: `UT-VOI-TestAudioRecorder_StartStop`

**Package**: `voice`

**File**: `internal/tools/voice/voice_test.go:93`

**Description**: TestAudioRecorder_StartStop tests recording start and stop

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Fatalf
5. Call Background
   ... and 5 more steps

---

### TestAuditLogger_LogAndQuery

**ID**: `UT-CON-TestAuditLogger_LogAndQuery`

**Package**: `confirmation`

**File**: `internal/tools/confirmation/confirmation_test.go:259`

**Description**: Test 9: AuditLogger - Log and query

**Test Steps**:
1. Call Now
2. Call Background
3. Call Log
4. Call Fatalf
5. Call Query
   ... and 3 more steps

---

### TestAuditLogger_QueryFiltering

**ID**: `UT-CON-TestAuditLogger_QueryFiltering`

**Package**: `confirmation`

**File**: `internal/tools/confirmation/confirmation_test.go:297`

**Description**: Test 10: AuditLogger - Query filtering

**Test Steps**:
1. Call Background
2. Call Now
3. Call Now
4. Call Now
5. Call Log
   ... and 5 more steps

---

### TestAuditStorage_Clear

**ID**: `UT-CON-TestAuditStorage_Clear`

**Package**: `confirmation`

**File**: `internal/tools/confirmation/confirmation_test.go:632`

**Description**: Test 20: Audit storage - Clear

**Test Steps**:
1. Call Background
2. Call Store
3. Call Fatalf
4. Call Clear
5. Call Fatalf
   ... and 3 more steps

---

### TestAuthService_GenerateJWT

**ID**: `UT-AUT-TestAuthService_GenerateJWT`

**Package**: `auth`

**File**: `internal/auth/auth_test.go:163`

**Description**: No description provided

**Test Steps**:
1. Call New
2. Call GenerateJWT
3. Call NoError
4. Call NotEmpty
5. Call VerifyJWT
   ... and 4 more steps

---

### TestAuthService_VerifyJWT

**ID**: `UT-AUT-TestAuthService_VerifyJWT`

**Package**: `auth`

**File**: `internal/auth/auth_test.go:183`

**Description**: No description provided

**Test Steps**:
1. Call VerifyJWT
2. Call Error

---

### TestAuthService_generateSessionToken

**ID**: `UT-AUT-TestAuthService_generateSessionToken`

**Package**: `auth`

**File**: `internal/auth/auth_test.go:194`

**Description**: No description provided

**Test Steps**:
1. Call generateSessionToken
2. Call NoError
3. Call NotEmpty
4. Call Len

---

### TestAuthService_hashPassword

**ID**: `UT-AUT-TestAuthService_hashPassword`

**Package**: `auth`

**File**: `internal/auth/auth_test.go:145`

**Description**: No description provided

**Test Steps**:
1. Call hashPassword
2. Call NoError
3. Call NotEmpty
4. Call NotEqual
5. Call verifyPassword
   ... and 3 more steps

---

### TestAuthService_validateRegistration

**ID**: `UT-AUT-TestAuthService_validateRegistration`

**Package**: `auth`

**File**: `internal/auth/auth_test.go:93`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call validateRegistration
3. Call Error
4. Call NoError

---

### TestAutoCommit

**ID**: `UT-GIT-TestAutoCommit`

**Package**: `git`

**File**: `internal/tools/git/git_test.go:149`

**Description**: TestAutoCommit tests the auto-commit functionality

**Test Steps**:
1. Call Run
2. Call Join
3. Call WriteFile
4. Call Fatal
5. Call Command
   ... and 5 more steps

---

### TestAutoSwitch

**ID**: `UT-VIS-TestAutoSwitch`

**Package**: `vision`

**File**: `internal/llm/vision/vision_test.go:341`

**Description**: TestAutoSwitch tests automatic model switching

**Test Steps**:
1. Call Fatalf
2. Call Background
3. Call Lock
4. Call Unlock
5. Call Run
   ... and 5 more steps

---

### TestAzureProvider_Close

**ID**: `UT-LLM-TestAzureProvider_Close`

**Package**: `llm`

**File**: `internal/llm/azure_provider_test.go:626`

**Description**: Test 18: Close provider

**Test Steps**:
1. Call NoError
2. Call Close
3. Call NoError

---

### TestAzureProvider_DefaultMaxTokens

**ID**: `UT-LLM-TestAzureProvider_DefaultMaxTokens`

**Package**: `llm`

**File**: `internal/llm/azure_provider_test.go:719`

**Description**: Test 20: Default max tokens

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Decode
4. Call NewDecoder
5. Call Equal
   ... and 5 more steps

---

### TestAzureProvider_DeploymentMapping_Explicit

**ID**: `UT-LLM-TestAzureProvider_DeploymentMapping_Explicit`

**Package**: `llm`

**File**: `internal/llm/azure_provider_test.go:90`

**Description**: Test 4: Deployment mapping - explicit mapping

**Test Steps**:
1. Call NoError
2. Call Equal
3. Call resolveDeployment
4. Call Equal
5. Call resolveDeployment

---

### TestAzureProvider_DeploymentMapping_Fallback

**ID**: `UT-LLM-TestAzureProvider_DeploymentMapping_Fallback`

**Package**: `llm`

**File**: `internal/llm/azure_provider_test.go:111`

**Description**: Test 5: Deployment mapping - fallback to model name

**Test Steps**:
1. Call NoError
2. Call Equal
3. Call resolveDeployment
4. Call Equal
5. Call resolveDeployment

---

### TestAzureProvider_DeploymentMapping_JSON

**ID**: `UT-LLM-TestAzureProvider_DeploymentMapping_JSON`

**Package**: `llm`

**File**: `internal/llm/azure_provider_test.go:129`

**Description**: Test 6: Deployment mapping from JSON string

**Test Steps**:
1. Call NoError
2. Call Equal
3. Call resolveDeployment

---

### TestAzureProvider_GenerateStream

**ID**: `UT-LLM-TestAzureProvider_GenerateStream`

**Package**: `llm`

**File**: `internal/llm/azure_provider_test.go:337`

**Description**: Test 11: Streaming generation

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Equal
4. Call Get
5. Call Set
   ... and 5 more steps

---

### TestAzureProvider_Generate_APIKey

**ID**: `UT-LLM-TestAzureProvider_Generate_APIKey`

**Package**: `llm`

**File**: `internal/llm/azure_provider_test.go:146`

**Description**: Test 7: Basic generation with API key

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Equal
4. Call Contains
5. Call Contains
   ... and 5 more steps

---

### TestAzureProvider_Generate_ContentFilter

**ID**: `UT-LLM-TestAzureProvider_Generate_ContentFilter`

**Package**: `llm`

**File**: `internal/llm/azure_provider_test.go:211`

**Description**: Test 8: Generation with content filtering error

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call WriteHeader
4. Call Encode
5. Call NewEncoder
   ... and 5 more steps

---

### TestAzureProvider_Generate_DeploymentNotFound

**ID**: `UT-LLM-TestAzureProvider_Generate_DeploymentNotFound`

**Package**: `llm`

**File**: `internal/llm/azure_provider_test.go:295`

**Description**: Test 10: Generation with deployment not found

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call WriteHeader
4. Call Encode
5. Call NewEncoder
   ... and 5 more steps

---

### TestAzureProvider_Generate_RateLimit

**ID**: `UT-LLM-TestAzureProvider_Generate_RateLimit`

**Package**: `llm`

**File**: `internal/llm/azure_provider_test.go:253`

**Description**: Test 9: Generation with rate limit error

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call WriteHeader
4. Call Encode
5. Call NewEncoder
   ... and 5 more steps

---

### TestAzureProvider_GetHealth_Failure

**ID**: `UT-LLM-TestAzureProvider_GetHealth_Failure`

**Package**: `llm`

**File**: `internal/llm/azure_provider_test.go:601`

**Description**: Test 17: GetHealth failure

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call WriteHeader
4. Call Write
5. Call Close
   ... and 5 more steps

---

### TestAzureProvider_GetHealth_Success

**ID**: `UT-LLM-TestAzureProvider_GetHealth_Success`

**Package**: `llm`

**File**: `internal/llm/azure_provider_test.go:553`

**Description**: Test 16: GetHealth success

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Unix
4. Call Now
5. Call Set
   ... and 5 more steps

---

### TestAzureProvider_IsAvailable

**ID**: `UT-LLM-TestAzureProvider_IsAvailable`

**Package**: `llm`

**File**: `internal/llm/azure_provider_test.go:533`

**Description**: Test 15: IsAvailable

**Test Steps**:
1. Call NoError
2. Call True
3. Call IsAvailable
4. Call Background
5. Call False
   ... and 2 more steps

---

### TestAzureProvider_ModelsAndCapabilities

**ID**: `UT-LLM-TestAzureProvider_ModelsAndCapabilities`

**Package**: `llm`

**File**: `internal/llm/azure_provider_test.go:495`

**Description**: Test 14: Models and capabilities

**Test Steps**:
1. Call NoError
2. Call GetModels
3. Call NotEmpty
4. Call Equal
5. Call True
   ... and 5 more steps

---

### TestAzureProvider_NewWithAPIKey

**ID**: `UT-LLM-TestAzureProvider_NewWithAPIKey`

**Package**: `llm`

**File**: `internal/llm/azure_provider_test.go:36`

**Description**: Test 1: Provider initialization with API key

**Test Steps**:
1. Call NoError
2. Call NotNil
3. Call Equal
4. Call Equal
5. Call Equal
   ... and 1 more steps

---

### TestAzureProvider_NewWithEntraID

**ID**: `UT-LLM-TestAzureProvider_NewWithEntraID`

**Package**: `llm`

**File**: `internal/llm/azure_provider_test.go:56`

**Description**: Test 2: Provider initialization with Entra ID

**Test Steps**:
1. Call Contains
2. Call Error

---

### TestAzureProvider_NewWithoutEndpoint

**ID**: `UT-LLM-TestAzureProvider_NewWithoutEndpoint`

**Package**: `llm`

**File**: `internal/llm/azure_provider_test.go:77`

**Description**: Test 3: Provider initialization without endpoint

**Test Steps**:
1. Call Error
2. Call Contains
3. Call Error

---

### TestAzureProvider_TypeAndName

**ID**: `UT-LLM-TestAzureProvider_TypeAndName`

**Package**: `llm`

**File**: `internal/llm/azure_provider_test.go:478`

**Description**: Test 13: Provider type and name

**Test Steps**:
1. Call NoError
2. Call Equal
3. Call GetType
4. Call Equal
5. Call GetName

---

### TestAzureProvider_WithTools

**ID**: `UT-LLM-TestAzureProvider_WithTools`

**Package**: `llm`

**File**: `internal/llm/azure_provider_test.go:643`

**Description**: Test 19: Tool support in request

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Decode
4. Call NewDecoder
5. Call NotEmpty
   ... and 5 more steps

---

### TestBackupManager_BackupRestore

**ID**: `UT-MUL-TestBackupManager_BackupRestore`

**Package**: `multiedit`

**File**: `internal/tools/multiedit/multiedit_test.go:47`

**Description**: Test 2: Backup and Restore

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call Join
4. Call WriteFile
5. Call NoError
   ... and 5 more steps

---

### TestBackupManager_Cleanup

**ID**: `UT-MUL-TestBackupManager_Cleanup`

**Package**: `multiedit`

**File**: `internal/tools/multiedit/multiedit_test.go:344`

**Description**: Test 12: Backup Cleanup

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call Join
4. Call WriteFile
5. Call Backup
   ... and 5 more steps

---

### TestBackupManager_CompressedBackup

**ID**: `UT-MUL-TestBackupManager_CompressedBackup`

**Package**: `multiedit`

**File**: `internal/tools/multiedit/multiedit_test.go:77`

**Description**: Test 3: Compressed Backup

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call Join
4. Call WriteFile
5. Call NoError
   ... and 5 more steps

---

### TestBaseAgentCanHandle

**ID**: `UT-AGE-TestBaseAgentCanHandle`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:113`

**Description**: No description provided

**Test Steps**:
1. Call CanHandle
2. Call Error
3. Call NewTask
4. Call CanHandle
5. Call Error
   ... and 5 more steps

---

### TestBaseAgentCapabilityMatching

**ID**: `UT-AGE-TestBaseAgentCapabilityMatching`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:208`

**Description**: No description provided

**Test Steps**:
1. Call NewTask
2. Call CanHandle
3. Call Error
4. Call NewTask
5. Call CanHandle
   ... and 4 more steps

---

### TestBaseAgentConcurrentCapabilityChecks

**ID**: `UT-AGE-TestBaseAgentConcurrentCapabilityChecks`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:753`

**Description**: No description provided

**Test Steps**:
1. Call NewTask
2. Call NewTask
3. Call CanHandle
4. Call Error
5. Call CanHandle
   ... and 1 more steps

---

### TestBaseAgentConcurrentErrorCounting

**ID**: `UT-AGE-TestBaseAgentConcurrentErrorCounting`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:677`

**Description**: No description provided

**Test Steps**:
1. Call IncrementTaskCount
2. Call IncrementErrorCount
3. Call Health
4. Call Errorf
5. Call Errorf

---

### TestBaseAgentConcurrentHealthChecks

**ID**: `UT-AGE-TestBaseAgentConcurrentHealthChecks`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:799`

**Description**: No description provided

**Test Steps**:
1. Call Add
2. Call Done
3. Call Health
4. Call ID
5. Call Errorf
   ... and 5 more steps

---

### TestBaseAgentConcurrentStatusChanges

**ID**: `UT-AGE-TestBaseAgentConcurrentStatusChanges`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:714`

**Description**: No description provided

**Test Steps**:
1. Call SetStatus
2. Call Status
3. Call Errorf

---

### TestBaseAgentConcurrentTaskCounting

**ID**: `UT-AGE-TestBaseAgentConcurrentTaskCounting`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:648`

**Description**: No description provided

**Test Steps**:
1. Call IncrementTaskCount
2. Call Health
3. Call Errorf

---

### TestBaseAgentEmptyConfig

**ID**: `UT-AGE-TestBaseAgentEmptyConfig`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:347`

**Description**: No description provided

**Test Steps**:
1. Call ID
2. Call Errorf
3. Call ID
4. Call Status
5. Call Errorf
   ... and 5 more steps

---

### TestBaseAgentErrorRateCalculation

**ID**: `UT-AGE-TestBaseAgentErrorRateCalculation`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:283`

**Description**: No description provided

**Test Steps**:
1. Call Health
2. Call Errorf
3. Call IncrementTaskCount
4. Call Health
5. Call Errorf
   ... and 3 more steps

---

### TestBaseAgentHealth

**ID**: `UT-AGE-TestBaseAgentHealth`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:159`

**Description**: No description provided

**Test Steps**:
1. Call Health
2. Call ID
3. Call Errorf
4. Call ID
5. Call Error
   ... and 5 more steps

---

### TestBaseAgentHealthWithOperations

**ID**: `UT-AGE-TestBaseAgentHealthWithOperations`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:246`

**Description**: No description provided

**Test Steps**:
1. Call Health
2. Call Error
3. Call IncrementTaskCount
4. Call IncrementTaskCount
5. Call IncrementTaskCount
   ... and 5 more steps

---

### TestBaseAgentStatusManagement

**ID**: `UT-AGE-TestBaseAgentStatusManagement`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:47`

**Description**: No description provided

**Test Steps**:
1. Call SetStatus
2. Call Status
3. Call Errorf
4. Call Status
5. Call SetStatus
   ... and 5 more steps

---

### TestBaseAgentStatusSequence

**ID**: `UT-AGE-TestBaseAgentStatusSequence`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:317`

**Description**: No description provided

**Test Steps**:
1. Call SetStatus
2. Call Status
3. Call Errorf
4. Call Status
5. Call SetStatus
   ... and 4 more steps

---

### TestBaseAgentTaskCounters

**ID**: `UT-AGE-TestBaseAgentTaskCounters`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:73`

**Description**: No description provided

**Test Steps**:
1. Call Health
2. Call Errorf
3. Call Errorf
4. Call IncrementTaskCount
5. Call IncrementTaskCount
   ... and 5 more steps

---

### TestBashPolicy_BlockSystemPaths

**ID**: `UT-CON-TestBashPolicy_BlockSystemPaths`

**Package**: `confirmation`

**File**: `internal/tools/confirmation/confirmation_test.go:576`

**Description**: Test 18: BashPolicy - Block system paths

**Test Steps**:
1. Call SetPolicy
2. Call Fatalf
3. Call Evaluate
4. Call Fatalf
5. Call Errorf

---

### TestBedrockProvider_BuildClaudeRequest

**ID**: `UT-LLM-TestBedrockProvider_BuildClaudeRequest`

**Package**: `llm`

**File**: `internal/llm/bedrock_provider_test.go:686`

**Description**: No description provided

**Test Steps**:
1. Call New
2. Call buildClaudeRequest
3. Call NoError
4. Call Unmarshal
5. Call NoError
   ... and 5 more steps

---

### TestBedrockProvider_BuildTitanRequest

**ID**: `UT-LLM-TestBedrockProvider_BuildTitanRequest`

**Package**: `llm`

**File**: `internal/llm/bedrock_provider_test.go:716`

**Description**: No description provided

**Test Steps**:
1. Call New
2. Call buildTitanRequest
3. Call NoError
4. Call Unmarshal
5. Call NoError
   ... and 4 more steps

---

### TestBedrockProvider_Close

**ID**: `UT-LLM-TestBedrockProvider_Close`

**Package**: `llm`

**File**: `internal/llm/bedrock_provider_test.go:823`

**Description**: No description provided

**Test Steps**:
1. Call Close
2. Call NoError

---

### TestBedrockProvider_CombineMessagesToPrompt

**ID**: `UT-LLM-TestBedrockProvider_CombineMessagesToPrompt`

**Package**: `llm`

**File**: `internal/llm/bedrock_provider_test.go:772`

**Description**: No description provided

**Test Steps**:
1. Call combineMessagesToPrompt
2. Call Contains
3. Call Contains
4. Call Contains
5. Call Contains

---

### TestBedrockProvider_ConvertTools

**ID**: `UT-LLM-TestBedrockProvider_ConvertTools`

**Package**: `llm`

**File**: `internal/llm/bedrock_provider_test.go:790`

**Description**: No description provided

**Test Steps**:
1. Call convertToolsToAnthropic
2. Call Len
3. Call Equal
4. Call Equal
5. Call convertToolsToCohere
   ... and 2 more steps

---

### TestBedrockProvider_CrossRegionInference

**ID**: `UT-LLM-TestBedrockProvider_CrossRegionInference`

**Package**: `llm`

**File**: `internal/llm/bedrock_provider_test.go:561`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call ToString
3. Call Marshal
4. Call String
5. Call New
   ... and 4 more steps

---

### TestBedrockProvider_ErrorHandling

**ID**: `UT-LLM-TestBedrockProvider_ErrorHandling`

**Package**: `llm`

**File**: `internal/llm/bedrock_provider_test.go:614`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call New
3. Call Generate
4. Call Background
5. Call Error
   ... and 1 more steps

---

### TestBedrockProvider_GenerateClaude

**ID**: `UT-LLM-TestBedrockProvider_GenerateClaude`

**Package**: `llm`

**File**: `internal/llm/bedrock_provider_test.go:210`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call ToString
3. Call Equal
4. Call ToString
5. Call Marshal
   ... and 5 more steps

---

### TestBedrockProvider_GenerateCommand

**ID**: `UT-LLM-TestBedrockProvider_GenerateCommand`

**Package**: `llm`

**File**: `internal/llm/bedrock_provider_test.go:378`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call ToString
3. Call Marshal
4. Call String
5. Call New
   ... and 5 more steps

---

### TestBedrockProvider_GenerateJurassic

**ID**: `UT-LLM-TestBedrockProvider_GenerateJurassic`

**Package**: `llm`

**File**: `internal/llm/bedrock_provider_test.go:321`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call ToString
3. Call Marshal
4. Call String
5. Call New
   ... and 5 more steps

---

### TestBedrockProvider_GenerateLlama

**ID**: `UT-LLM-TestBedrockProvider_GenerateLlama`

**Package**: `llm`

**File**: `internal/llm/bedrock_provider_test.go:432`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call ToString
3. Call Marshal
4. Call String
5. Call New
   ... and 5 more steps

---

### TestBedrockProvider_GenerateStream

**ID**: `UT-LLM-TestBedrockProvider_GenerateStream`

**Package**: `llm`

**File**: `internal/llm/bedrock_provider_test.go:829`

**Description**: No description provided

**Test Steps**:
1. Call Skip

---

### TestBedrockProvider_GenerateTitan

**ID**: `UT-LLM-TestBedrockProvider_GenerateTitan`

**Package**: `llm`

**File**: `internal/llm/bedrock_provider_test.go:272`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call ToString
3. Call Marshal
4. Call String
5. Call New
   ... and 5 more steps

---

### TestBedrockProvider_GenerateWithTools

**ID**: `UT-LLM-TestBedrockProvider_GenerateWithTools`

**Package**: `llm`

**File**: `internal/llm/bedrock_provider_test.go:477`

**Description**: No description provided

**Test Steps**:
1. Call Unmarshal
2. Call NoError
3. Call NotEmpty
4. Call Equal
5. Call Marshal
   ... and 5 more steps

---

### TestBedrockProvider_GetCapabilities

**ID**: `UT-LLM-TestBedrockProvider_GetCapabilities`

**Package**: `llm`

**File**: `internal/llm/bedrock_provider_test.go:167`

**Description**: No description provided

**Test Steps**:
1. Call GetCapabilities
2. Call NotEmpty
3. Call True
4. Call True
5. Call True
   ... and 2 more steps

---

### TestBedrockProvider_GetModelFamily

**ID**: `UT-LLM-TestBedrockProvider_GetModelFamily`

**Package**: `llm`

**File**: `internal/llm/bedrock_provider_test.go:187`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call getModelFamily
3. Call Equal

---

### TestBedrockProvider_GetModels

**ID**: `UT-LLM-TestBedrockProvider_GetModels`

**Package**: `llm`

**File**: `internal/llm/bedrock_provider_test.go:140`

**Description**: No description provided

**Test Steps**:
1. Call GetModels
2. Call NotEmpty
3. Call Equal
4. Call Greater
5. Call NotEmpty
   ... and 5 more steps

---

### TestBedrockProvider_GetName

**ID**: `UT-LLM-TestBedrockProvider_GetName`

**Package**: `llm`

**File**: `internal/llm/bedrock_provider_test.go:135`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call GetName

---

### TestBedrockProvider_GetType

**ID**: `UT-LLM-TestBedrockProvider_GetType`

**Package**: `llm`

**File**: `internal/llm/bedrock_provider_test.go:130`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call GetType

---

### TestBedrockProvider_HandleBedrockError

**ID**: `UT-LLM-TestBedrockProvider_HandleBedrockError`

**Package**: `llm`

**File**: `internal/llm/bedrock_provider_test.go:836`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call handleBedrockError
3. Call NoError
4. Call Error
5. Call ErrorIs

---

### TestBedrockProvider_Integration

**ID**: `UT-LLM-TestBedrockProvider_Integration`

**Package**: `llm`

**File**: `internal/llm/bedrock_provider_test.go:890`

**Description**: Integration test - requires actual AWS credentials

**Test Steps**:
1. Call Short
2. Call Skip
3. Call Skipf
4. Call True
5. Call IsAvailable
   ... and 5 more steps

---

### TestBedrockProvider_IsAvailable

**ID**: `UT-LLM-TestBedrockProvider_IsAvailable`

**Package**: `llm`

**File**: `internal/llm/bedrock_provider_test.go:742`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call IsAvailable
3. Call Background
4. Call Equal

---

### TestBlockedCommand

**ID**: `UT-SHE-TestBlockedCommand`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:101`

**Description**: TestBlockedCommand tests that blocked commands are rejected

**Test Steps**:
1. Call Run
2. Call Execute
3. Call Background
4. Call Error
5. Call ErrorAs

---

### TestBrowserActions

**ID**: `UT-BRO-TestBrowserActions`

**Package**: `browser`

**File**: `internal/tools/browser/browser_test.go:178`

**Description**: TestBrowserActions tests browser action execution

**Test Steps**:
1. Call WithTimeout
2. Call Background
3. Call Launch
4. Call Skipf
5. Call Close
   ... and 5 more steps

---

### TestBrowserLaunch

**ID**: `UT-BRO-TestBrowserLaunch`

**Package**: `browser`

**File**: `internal/tools/browser/browser_test.go:79`

**Description**: TestBrowserLaunch tests browser launch functionality

**Test Steps**:
1. Call Run
2. Call WithTimeout
3. Call Background
4. Call Launch
5. Call Skipf
   ... and 5 more steps

---

### TestBrowserSession

**ID**: `UT-BRO-TestBrowserSession`

**Package**: `browser`

**File**: `internal/tools/browser/browser_test.go:430`

**Description**: TestBrowserSession tests browser session management

**Test Steps**:
1. Call WithTimeout
2. Call Background
3. Call Skipf
4. Call Close
5. Call Run
   ... and 5 more steps

---

### TestBrowserTools

**ID**: `UT-BRO-TestBrowserTools`

**Package**: `browser`

**File**: `internal/tools/browser/browser_test.go:352`

**Description**: TestBrowserTools tests the unified BrowserTools interface

**Test Steps**:
1. Call NotNil
2. Call Run
3. Call WithTimeout
4. Call Background
5. Call LaunchBrowser
   ... and 5 more steps

---

### TestBudgetStatus

**ID**: `UT-LLM-TestBudgetStatus`

**Package**: `llm`

**File**: `internal/llm/provider_features_test.go:217`

**Description**: TestBudgetStatus tests budget status reporting

**Test Steps**:
1. Call New
2. Call TrackRequest
3. Call GetBudgetStatus
4. Call NotNil
5. Call Equal
   ... and 5 more steps

---

### TestBudgetValidation

**ID**: `UT-LLM-TestBudgetValidation`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:83`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Equal

---

### TestCORSMiddleware

**ID**: `UT-SER-TestCORSMiddleware`

**Package**: `server`

**File**: `internal/server/server_test.go:74`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call New
3. Call Use
4. Call GET
5. Call JSON
   ... and 5 more steps

---

### TestCacheCleanup

**ID**: `UT-REP-TestCacheCleanup`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:775`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call Set
4. Call Set
5. Call Sleep
   ... and 4 more steps

---

### TestCacheConfigStrategies

**ID**: `UT-LLM-TestCacheConfigStrategies`

**Package**: `llm`

**File**: `internal/llm/provider_features_test.go:56`

**Description**: TestCacheConfigStrategies tests cache configuration strategies

**Test Steps**:
1. Call Run
2. Call Equal

---

### TestCacheControl

**ID**: `UT-LLM-TestCacheControl`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:21`

**Description**: TestCacheControl tests basic CacheControl structure

**Test Steps**:
1. Call Run
2. Call Equal
3. Call Nil

---

### TestCacheEnabled

**ID**: `UT-REP-TestCacheEnabled`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:318`

**Description**: Test cache functionality

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call extractFileSymbols
4. Call Fatalf
5. Call extractFileSymbols
   ... and 2 more steps

---

### TestCacheExpiration

**ID**: `UT-REP-TestCacheExpiration`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:740`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call Set
4. Call Sleep
5. Call Get
   ... and 1 more steps

---

### TestCacheGetOrCompute

**ID**: `UT-REP-TestCacheGetOrCompute`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:815`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call GetOrCompute
4. Call Error
5. Call Error
   ... and 4 more steps

---

### TestCacheGetSet

**ID**: `UT-REP-TestCacheGetSet`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:695`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call Set
4. Call Sleep
5. Call Get
   ... and 2 more steps

---

### TestCacheGetStats

**ID**: `UT-REP-TestCacheGetStats`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:794`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call Set
4. Call Set
5. Call Sleep
   ... and 3 more steps

---

### TestCacheHas

**ID**: `UT-REP-TestCacheHas`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:847`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call Has
4. Call Error
5. Call Set
   ... and 3 more steps

---

### TestCacheInvalidate

**ID**: `UT-REP-TestCacheInvalidate`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:719`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call Set
4. Call Sleep
5. Call Invalidate
   ... and 3 more steps

---

### TestCacheKeys

**ID**: `UT-REP-TestCacheKeys`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:865`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call Set
4. Call Set
5. Call Sleep
   ... and 2 more steps

---

### TestCacheManager

**ID**: `UT-FIL-TestCacheManager`

**Package**: `filesystem`

**File**: `internal/tools/filesystem/filesystem_test.go:671`

**Description**: TestCacheManager tests cache functionality

**Test Steps**:
1. Call Fatal
2. Call Now
3. Call Run
4. Call Set
5. Call Get
   ... and 5 more steps

---

### TestCacheManager_Clear

**ID**: `UT-WEB-TestCacheManager_Clear`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:480`

**Description**: Test 20: Cache clear functionality

**Test Steps**:
1. Call TempDir
2. Call Set
3. Call Set
4. Call Set
5. Call Get
   ... and 5 more steps

---

### TestCacheManager_Expiration

**ID**: `UT-WEB-TestCacheManager_Expiration`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:222`

**Description**: Test 10: Cache expiration

**Test Steps**:
1. Call TempDir
2. Call Set
3. Call Get
4. Call True
5. Call Sleep
   ... and 3 more steps

---

### TestCacheManager_HitAndMiss

**ID**: `UT-WEB-TestCacheManager_HitAndMiss`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:198`

**Description**: Test 9: Cache hit and miss

**Test Steps**:
1. Call TempDir
2. Call Get
3. Call False
4. Call Equal
5. Call Load
   ... and 5 more steps

---

### TestCacheManager_Stats

**ID**: `UT-WEB-TestCacheManager_Stats`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:575`

**Description**: Test 25: Cache stats

**Test Steps**:
1. Call TempDir
2. Call GetStats
3. Call NotNil
4. Call Get
5. Call Get
   ... and 5 more steps

---

### TestCacheMetricsAverageSavings

**ID**: `UT-LLM-TestCacheMetricsAverageSavings`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:704`

**Description**: TestCacheMetricsAverageSavings tests average savings calculation

**Test Steps**:
1. Call UpdateMetrics
2. Call InDelta
3. Call UpdateMetrics
4. Call InDelta
5. Call UpdateMetrics
   ... and 1 more steps

---

### TestCacheMetricsCacheHitRate

**ID**: `UT-LLM-TestCacheMetricsCacheHitRate`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:652`

**Description**: TestCacheMetricsCacheHitRate tests cache hit rate calculation

**Test Steps**:
1. Call Run
2. Call UpdateMetrics
3. Call InDelta

---

### TestCacheMetricsTotals

**ID**: `UT-LLM-TestCacheMetricsTotals`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:730`

**Description**: TestCacheMetricsTotals tests total accumulation

**Test Steps**:
1. Call UpdateMetrics
2. Call Equal
3. Call Equal
4. Call InDelta

---

### TestCacheMetricsTracking

**ID**: `UT-LLM-TestCacheMetricsTracking`

**Package**: `llm`

**File**: `internal/llm/provider_features_test.go:476`

**Description**: TestCacheMetricsTracking tests cache metrics accumulation

**Test Steps**:
1. Call UpdateMetrics
2. Call Equal
3. Call Equal
4. Call InDelta
5. Call Equal
   ... and 3 more steps

---

### TestCacheMetricsUpdateMetrics

**ID**: `UT-LLM-TestCacheMetricsUpdateMetrics`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:554`

**Description**: TestCacheMetricsUpdateMetrics tests updating cache metrics

**Test Steps**:
1. Call Run
2. Call UpdateMetrics
3. Call Equal
4. Call Equal
5. Call InDelta

---

### TestCacheSavingsCalculation

**ID**: `UT-LLM-TestCacheSavingsCalculation`

**Package**: `llm`

**File**: `internal/llm/provider_features_test.go:278`

**Description**: TestCacheSavingsCalculation tests cache cost savings calculation

**Test Steps**:
1. Call InDelta
2. Call InDelta
3. Call InDelta
4. Call InDelta

---

### TestCacheSavingsStructure

**ID**: `UT-LLM-TestCacheSavingsStructure`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:1007`

**Description**: TestCacheSavingsStructure tests the CacheSavings structure

**Test Steps**:
1. Call Equal
2. Call Equal
3. Call Equal
4. Call Equal
5. Call Equal
   ... and 1 more steps

---

### TestCacheSize

**ID**: `UT-REP-TestCacheSize`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:756`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call Size
4. Call Error
5. Call Set
   ... and 5 more steps

---

### TestCacheStatsStructure

**ID**: `UT-LLM-TestCacheStatsStructure`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:992`

**Description**: TestCacheStatsStructure tests the CacheStats structure

**Test Steps**:
1. Call Equal
2. Call Equal
3. Call Equal
4. Call Equal

---

### TestCacheStrategyAggressive

**ID**: `UT-LLM-TestCacheStrategyAggressive`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:350`

**Description**: TestCacheStrategyAggressive tests aggressive caching strategy

**Test Steps**:
1. Call Run
2. Call NotNil
3. Call Equal
4. Call Nil

---

### TestCacheStrategyContext

**ID**: `UT-LLM-TestCacheStrategyContext`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:272`

**Description**: TestCacheStrategyContext tests system + context caching strategy

**Test Steps**:
1. Call Run
2. Call NotNil
3. Call Equal
4. Call Nil

---

### TestCacheStrategyNone

**ID**: `UT-LLM-TestCacheStrategyNone`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:95`

**Description**: TestCacheStrategyNone tests that no caching is applied with CacheStrategyNone

**Test Steps**:
1. Call Len
2. Call Nil

---

### TestCacheStrategyNoneDisabled

**ID**: `UT-LLM-TestCacheStrategyNoneDisabled`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:115`

**Description**: TestCacheStrategyNoneDisabled tests that no caching is applied when disabled

**Test Steps**:
1. Call Len
2. Call Nil

---

### TestCacheStrategySystem

**ID**: `UT-LLM-TestCacheStrategySystem`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:135`

**Description**: TestCacheStrategySystem tests system-only caching strategy

**Test Steps**:
1. Call Run
2. Call Len
3. Call NotNil
4. Call Equal
5. Call Nil

---

### TestCacheStrategyTools

**ID**: `UT-LLM-TestCacheStrategyTools`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:204`

**Description**: TestCacheStrategyTools tests system + tools caching strategy

**Test Steps**:
1. Call Run
2. Call NotNil
3. Call Equal
4. Call Nil

---

### TestCacheableMessageStructure

**ID**: `UT-LLM-TestCacheableMessageStructure`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:972`

**Description**: TestCacheableMessageStructure tests the CacheableMessage structure

**Test Steps**:
1. Call Equal
2. Call Equal
3. Call Equal
4. Call NotNil
5. Call Equal

---

### TestCalculateCacheSavings

**ID**: `UT-LLM-TestCalculateCacheSavings`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:415`

**Description**: TestCalculateCacheSavings tests cache savings calculations

**Test Steps**:
1. Call Run
2. Call InDelta
3. Call InDelta
4. Call Equal
5. Call Equal
   ... and 1 more steps

---

### TestCalculateCacheSavingsEdgeCases

**ID**: `UT-LLM-TestCalculateCacheSavingsEdgeCases`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:502`

**Description**: TestCalculateCacheSavingsEdgeCases tests edge cases for cache savings

**Test Steps**:
1. Call Run
2. Call NotNil

---

### TestCalculateChecksum

**ID**: `UT-MAP-TestCalculateChecksum`

**Package**: `mapping`

**File**: `internal/tools/mapping/mapping_test.go:382`

**Description**: TestCalculateChecksum tests checksum calculation

**Test Steps**:
1. Call NotEqual
2. Call Equal
3. Call Len

---

### TestCalculateReasoningCost_Claude_Sonnet

**ID**: `UT-LLM-TestCalculateReasoningCost_Claude_Sonnet`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:434`

**Description**: No description provided

**Test Steps**:
1. Call Greater
2. Call Greater
3. Call InDelta
4. Call InDelta
5. Call InDelta

---

### TestCalculateReasoningCost_DeepSeek_R1

**ID**: `UT-LLM-TestCalculateReasoningCost_DeepSeek_R1`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:455`

**Description**: No description provided

**Test Steps**:
1. Call Greater
2. Call Greater
3. Call Equal
4. Call Less

---

### TestCalculateReasoningCost_OpenAI_O1

**ID**: `UT-LLM-TestCalculateReasoningCost_OpenAI_O1`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:413`

**Description**: Test CalculateReasoningCost

**Test Steps**:
1. Call Greater
2. Call Greater
3. Call Equal
4. Call InDelta
5. Call InDelta

---

### TestChannelRateLimits

**ID**: `UT-NOT-TestChannelRateLimits`

**Package**: `notification`

**File**: `internal/notification/ratelimit_test.go:307`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call NotNil
3. Call NotNil
4. Call NotNil
5. Call NotNil

---

### TestCheckBudget_DailyCostLimit

**ID**: `UT-LLM-TestCheckBudget_DailyCostLimit`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:275`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call TrackRequest
3. Call TrackRequest
4. Call CheckBudget
5. Call Error
   ... and 2 more steps

---

### TestCheckBudget_NewSession

**ID**: `UT-LLM-TestCheckBudget_NewSession`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:170`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call CheckBudget
3. Call NoError

---

### TestCheckBudget_PerRequestLimit

**ID**: `UT-LLM-TestCheckBudget_PerRequestLimit`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:183`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call Run
3. Call CheckBudget
4. Call Error
5. Call Contains
   ... and 2 more steps

---

### TestCheckBudget_SessionCostLimit

**ID**: `UT-LLM-TestCheckBudget_SessionCostLimit`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:247`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call TrackRequest
3. Call CheckBudget
4. Call NoError
5. Call CheckBudget
   ... and 3 more steps

---

### TestCheckBudget_SessionTokenLimit

**ID**: `UT-LLM-TestCheckBudget_SessionTokenLimit`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:219`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call TrackRequest
3. Call CheckBudget
4. Call NoError
5. Call CheckBudget
   ... and 3 more steps

---

### TestCheckBudget_WarningThreshold_Cost

**ID**: `UT-LLM-TestCheckBudget_WarningThreshold_Cost`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:327`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call TrackRequest
3. Call CheckBudget
4. Call Error
5. Call Contains
   ... and 1 more steps

---

### TestCheckBudget_WarningThreshold_Tokens

**ID**: `UT-LLM-TestCheckBudget_WarningThreshold_Tokens`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:303`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call TrackRequest
3. Call CheckBudget
4. Call Error
5. Call Contains
   ... and 1 more steps

---

### TestChromeDiscovery

**ID**: `UT-BRO-TestChromeDiscovery`

**Package**: `browser`

**File**: `internal/tools/browser/browser_test.go:14`

**Description**: TestChromeDiscovery tests Chrome discovery functionality

**Test Steps**:
1. Call Run
2. Call FindChrome
3. Call Skip
4. Call NotEmpty
5. Call Stat
   ... and 5 more steps

---

### TestCircuitBreakerCallWhenOpen

**ID**: `UT-AGE-TestCircuitBreakerCallWhenOpen`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:112`

**Description**: No description provided

**Test Steps**:
1. Call recordFailure
2. Call recordFailure
3. Call Equal
4. Call GetState
5. Call Background
   ... and 4 more steps

---

### TestCircuitBreakerClosedToOpen

**ID**: `UT-AGE-TestCircuitBreakerClosedToOpen`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:27`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call GetState
3. Call recordFailure
4. Call Equal
5. Call GetState
   ... and 3 more steps

---

### TestCircuitBreakerConcurrency

**ID**: `UT-AGE-TestCircuitBreakerConcurrency`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:523`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call Call
3. Call Sleep
4. Call Equal
5. Call GetState

---

### TestCircuitBreakerEdgeCases

**ID**: `UT-AGE-TestCircuitBreakerEdgeCases`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:575`

**Description**: No description provided

**Test Steps**:
1. Call recordFailure
2. Call Equal
3. Call GetState
4. Call Sleep
5. Call Background
   ... and 4 more steps

---

### TestCircuitBreakerHalfOpen

**ID**: `UT-AGE-TestCircuitBreakerHalfOpen`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:44`

**Description**: No description provided

**Test Steps**:
1. Call recordFailure
2. Call recordFailure
3. Call Equal
4. Call GetState
5. Call Sleep
   ... and 5 more steps

---

### TestCircuitBreakerHalfOpenToClosed

**ID**: `UT-AGE-TestCircuitBreakerHalfOpenToClosed`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:88`

**Description**: No description provided

**Test Steps**:
1. Call recordFailure
2. Call recordFailure
3. Call Equal
4. Call GetState
5. Call Sleep
   ... and 5 more steps

---

### TestCircuitBreakerHalfOpenToOpen

**ID**: `UT-AGE-TestCircuitBreakerHalfOpenToOpen`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:67`

**Description**: No description provided

**Test Steps**:
1. Call recordFailure
2. Call recordFailure
3. Call Equal
4. Call GetState
5. Call Sleep
   ... and 5 more steps

---

### TestCircuitBreakerManagerGetOrCreate

**ID**: `UT-AGE-TestCircuitBreakerManagerGetOrCreate`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:452`

**Description**: No description provided

**Test Steps**:
1. Call GetOrCreate
2. Call NotNil
3. Call Equal
4. Call GetOrCreate
5. Call Equal
   ... and 4 more steps

---

### TestCircuitBreakerManagerGetState

**ID**: `UT-AGE-TestCircuitBreakerManagerGetState`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:488`

**Description**: No description provided

**Test Steps**:
1. Call GetState
2. Call Equal
3. Call GetOrCreate
4. Call recordFailure
5. Call recordFailure
   ... and 2 more steps

---

### TestCircuitBreakerManagerGetStats

**ID**: `UT-AGE-TestCircuitBreakerManagerGetStats`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:505`

**Description**: No description provided

**Test Steps**:
1. Call GetOrCreate
2. Call recordFailure
3. Call recordFailure
4. Call GetOrCreate
5. Call GetStats
   ... and 3 more steps

---

### TestCircuitBreakerManagerReset

**ID**: `UT-AGE-TestCircuitBreakerManagerReset`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:471`

**Description**: No description provided

**Test Steps**:
1. Call GetOrCreate
2. Call recordFailure
3. Call recordFailure
4. Call Equal
5. Call GetState
   ... and 3 more steps

---

### TestCircuitBreakerReset

**ID**: `UT-AGE-TestCircuitBreakerReset`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:130`

**Description**: No description provided

**Test Steps**:
1. Call recordFailure
2. Call recordFailure
3. Call Equal
4. Call GetState
5. Call Reset
   ... and 4 more steps

---

### TestCleanupOldSessions_AllCurrent

**ID**: `UT-LLM-TestCleanupOldSessions_AllCurrent`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:808`

**Description**: No description provided

**Test Steps**:
1. Call String
2. Call New
3. Call TrackRequest
4. Call CleanupOldSessions
5. Call Equal
   ... and 2 more steps

---

### TestCleanupOldSessions_MixedAges

**ID**: `UT-LLM-TestCleanupOldSessions_MixedAges`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:827`

**Description**: No description provided

**Test Steps**:
1. Call TrackRequest
2. Call TrackRequest
3. Call Lock
4. Call Add
5. Call Now
   ... and 5 more steps

---

### TestCleanupOldSessions_NoSessions

**ID**: `UT-LLM-TestCleanupOldSessions_NoSessions`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:800`

**Description**: No description provided

**Test Steps**:
1. Call CleanupOldSessions
2. Call Equal

---

### TestCleanupOldSessions_VariousThresholds

**ID**: `UT-LLM-TestCleanupOldSessions_VariousThresholds`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:860`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call TrackRequest
3. Call Lock
4. Call Add
5. Call Now
   ... and 5 more steps

---

### TestClient_Close

**ID**: `UT-RED-TestClient_Close`

**Package**: `redis`

**File**: `internal/redis/redis_test.go:103`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call Close
3. Call NoError
4. Call Close

---

### TestClient_Methods_Disabled

**ID**: `UT-RED-TestClient_Methods_Disabled`

**Package**: `redis`

**File**: `internal/redis/redis_test.go:39`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call NoError
3. Call Set
4. Call NoError
5. Call Del
   ... and 5 more steps

---

### TestCloseAllSessions

**ID**: `UT-MCP-TestCloseAllSessions`

**Package**: `mcp`

**File**: `internal/mcp/server_test.go:43`

**Description**: No description provided

**Test Steps**:
1. Call CloseAllSessions
2. Call Equal
3. Call GetSessionCount

---

### TestCodeEditorApplyEditIntegration

**ID**: `UT-EDI-TestCodeEditorApplyEditIntegration`

**Package**: `editor`

**File**: `internal/editor/editor_test.go:351`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Run
5. Call Join
   ... and 5 more steps

---

### TestCodeEditorBackup

**ID**: `UT-EDI-TestCodeEditorBackup`

**Package**: `editor`

**File**: `internal/editor/editor_test.go:154`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Join
5. Call WriteFile
   ... and 5 more steps

---

### TestCodeEditorConcurrentEdits

**ID**: `UT-EDI-TestCodeEditorConcurrentEdits`

**Package**: `editor`

**File**: `internal/editor/editor_test.go:211`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Join
5. Call WriteFile
   ... and 5 more steps

---

### TestCodeEditorSetFormat

**ID**: `UT-EDI-TestCodeEditorSetFormat`

**Package**: `editor`

**File**: `internal/editor/editor_test.go:44`

**Description**: No description provided

**Test Steps**:
1. Call Fatalf
2. Call Run
3. Call SetFormat
4. Call Error
5. Call Errorf
   ... and 3 more steps

---

### TestCodeEditorValidateEdit

**ID**: `UT-EDI-TestCodeEditorValidateEdit`

**Package**: `editor`

**File**: `internal/editor/editor_test.go:80`

**Description**: No description provided

**Test Steps**:
1. Call Fatalf
2. Call Run
3. Call ValidateEdit
4. Call Error
5. Call Errorf

---

### TestCodebaseMap

**ID**: `UT-MAP-TestCodebaseMap`

**Package**: `mapping`

**File**: `internal/tools/mapping/mapping_test.go:397`

**Description**: TestCodebaseMap tests CodebaseMap operations

**Test Steps**:
1. Call Run
2. Call AddFile
3. Call Equal
4. Call Equal
5. Call Equal
   ... and 5 more steps

---

### TestCodingAgentCollaborate

**ID**: `UT-TYP-TestCodingAgentCollaborate`

**Package**: `types`

**File**: `internal/agent/types/coding_agent_test.go:257`

**Description**: TestCodingAgentCollaborate tests collaboration with review agents

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call NewBaseAgent
5. Call Background
   ... and 5 more steps

---

### TestCodingAgentExecuteCreate

**ID**: `UT-TYP-TestCodingAgentExecuteCreate`

**Package**: `types`

**File**: `internal/agent/types/coding_agent_test.go:107`

**Description**: TestCodingAgentExecuteCreate tests code creation

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call NewTask
   ... and 5 more steps

---

### TestCodingAgentExecuteEdit

**ID**: `UT-TYP-TestCodingAgentExecuteEdit`

**Package**: `types`

**File**: `internal/agent/types/coding_agent_test.go:148`

**Description**: TestCodingAgentExecuteEdit tests code editing

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call NewTask
   ... and 4 more steps

---

### TestCodingAgentExecuteLLMError

**ID**: `UT-TYP-TestCodingAgentExecuteLLMError`

**Package**: `types`

**File**: `internal/agent/types/coding_agent_test.go:221`

**Description**: TestCodingAgentExecuteLLMError tests LLM generation error

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call NewTask
   ... and 5 more steps

---

### TestCodingAgentExecuteMissingRequirements

**ID**: `UT-TYP-TestCodingAgentExecuteMissingRequirements`

**Package**: `types`

**File**: `internal/agent/types/coding_agent_test.go:187`

**Description**: TestCodingAgentExecuteMissingRequirements tests error when requirements missing

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call NewTask
   ... and 5 more steps

---

### TestCodingAgentInitialize

**ID**: `UT-TYP-TestCodingAgentInitialize`

**Package**: `types`

**File**: `internal/agent/types/coding_agent_test.go:65`

**Description**: TestCodingAgentInitialize tests agent initialization

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call Initialize
   ... and 3 more steps

---

### TestCodingAgentShutdown

**ID**: `UT-TYP-TestCodingAgentShutdown`

**Package**: `types`

**File**: `internal/agent/types/coding_agent_test.go:86`

**Description**: TestCodingAgentShutdown tests agent shutdown

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call Shutdown
   ... and 3 more steps

---

### TestCodingAgentTaskMetrics

**ID**: `UT-TYP-TestCodingAgentTaskMetrics`

**Package**: `types`

**File**: `internal/agent/types/coding_agent_test.go:307`

**Description**: TestCodingAgentTaskMetrics tests task metrics recording

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call NewTask
   ... and 5 more steps

---

### TestColorToHex

**ID**: `UT-LOG-TestColorToHex`

**Package**: `logo`

**File**: `internal/logo/processor_test.go:23`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call Equal
3. Call Equal

---

### TestCommandInjectionPrevention

**ID**: `UT-SHE-TestCommandInjectionPrevention`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:346`

**Description**: TestCommandInjectionPrevention tests command injection prevention

**Test Steps**:
1. Call Run
2. Call Execute
3. Call Background
4. Call Error

---

### TestCommandTimeout

**ID**: `UT-SHE-TestCommandTimeout`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:48`

**Description**: TestCommandTimeout tests command timeout

**Test Steps**:
1. Call Execute
2. Call Background
3. Call NoError
4. Call True
5. Call True

---

### TestCommandWithArgs

**ID**: `UT-SHE-TestCommandWithArgs`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:609`

**Description**: TestCommandWithArgs tests command execution with arguments

**Test Steps**:
1. Call Execute
2. Call Background
3. Call NoError
4. Call Equal

---

### TestCommandWithEnvironment

**ID**: `UT-SHE-TestCommandWithEnvironment`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:64`

**Description**: TestCommandWithEnvironment tests command with environment variables

**Test Steps**:
1. Call Execute
2. Call Background
3. Call NoError
4. Call Equal
5. Call Contains

---

### TestCommandWithExitCode

**ID**: `UT-SHE-TestCommandWithExitCode`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:34`

**Description**: TestCommandWithExitCode tests command with non-zero exit code

**Test Steps**:
1. Call Execute
2. Call Background
3. Call NoError
4. Call Equal

---

### TestCommandWithWorkDir

**ID**: `UT-SHE-TestCommandWithWorkDir`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:82`

**Description**: TestCommandWithWorkDir tests command with working directory

**Test Steps**:
1. Call TempDir
2. Call Execute
3. Call Background
4. Call NoError
5. Call Equal
   ... and 1 more steps

---

### TestCompareSnapshots

**ID**: `UT-SNA-TestCompareSnapshots`

**Package**: `snapshots`

**File**: `internal/workflow/snapshots/snapshots_test.go:444`

**Description**: TestCompareSnapshots tests comparing two snapshots

**Test Steps**:
1. Call Fatalf
2. Call Background
3. Call CreateSnapshot
4. Call Fatalf
5. Call Command
   ... and 5 more steps

---

### TestCompressionCoordinator_Compress

**ID**: `UT-COM-TestCompressionCoordinator_Compress`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:416`

**Description**: Test 14: Compression Coordinator - Full Compression Flow

**Test Steps**:
1. Call Compress
2. Call Background
3. Call NoError
4. Call NotNil
5. Call Less
   ... and 4 more steps

---

### TestCompressionCoordinator_EstimateCompression

**ID**: `UT-COM-TestCompressionCoordinator_EstimateCompression`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:445`

**Description**: Test 15: Compression Coordinator - EstimateCompression

**Test Steps**:
1. Call EstimateCompression
2. Call NoError
3. Call Greater
4. Call Greater
5. Call Greater

---

### TestCompressionCoordinator_ShouldCompress

**ID**: `UT-COM-TestCompressionCoordinator_ShouldCompress`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:398`

**Description**: Test 13: Compression Coordinator - ShouldCompress

**Test Steps**:
1. Call ShouldCompress
2. Call False
3. Call Empty
4. Call ShouldCompress
5. Call True
   ... and 1 more steps

---

### TestCompressionCoordinator_Stats

**ID**: `UT-COM-TestCompressionCoordinator_Stats`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:606`

**Description**: Test 23: Compression Statistics

**Test Steps**:
1. Call GetStats
2. Call Equal
3. Call Compress
4. Call Background
5. Call NoError
   ... and 4 more steps

---

### TestCompression_PreserveSystemMessages

**ID**: `UT-COM-TestCompression_PreserveSystemMessages`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:581`

**Description**: Test 22: Compression with System Messages

**Test Steps**:
1. Call Compress
2. Call Background
3. Call NoError
4. Call Equal
5. Call True

---

### TestConcurrentExecution

**ID**: `UT-SHE-TestConcurrentExecution`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:210`

**Description**: TestConcurrentExecution tests concurrent command execution

**Test Steps**:
1. Call Add
2. Call Done
3. Call Sprintf
4. Call Sprintf
5. Call Execute
   ... and 5 more steps

---

### TestCondition_MatchesPathPattern

**ID**: `UT-CON-TestCondition_MatchesPathPattern`

**Package**: `confirmation`

**File**: `internal/tools/confirmation/confirmation_test.go:532`

**Description**: Test 16: Condition matching - Path pattern

**Test Steps**:
1. Call Matches
2. Call Error
3. Call Matches
4. Call Error

---

### TestCondition_MatchesRiskLevel

**ID**: `UT-CON-TestCondition_MatchesRiskLevel`

**Package**: `confirmation`

**File**: `internal/tools/confirmation/confirmation_test.go:554`

**Description**: Test 17: Condition matching - Risk level

**Test Steps**:
1. Call Matches
2. Call Error
3. Call Matches
4. Call Error

---

### TestConfig

**ID**: `UT-DAT-TestConfig`

**Package**: `database`

**File**: `internal/database/database_test.go:9`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call Equal
3. Call Equal
4. Call Equal
5. Call Equal
   ... and 1 more steps

---

### TestConfig

**ID**: `UT-GIT-TestConfig`

**Package**: `git`

**File**: `internal/tools/git/git_test.go:859`

**Description**: TestConfig tests configuration

**Test Steps**:
1. Call Run
2. Call Fatal
3. Call Errorf
4. Call Error
5. Call Error

---

### TestConfigValidation

**ID**: `UT-AUT-TestConfigValidation`

**Package**: `autonomy`

**File**: `internal/workflow/autonomy/autonomy_test.go:572`

**Description**: TestConfigValidation tests configuration validation

**Test Steps**:
1. Call Run
2. Call Validate
3. Call Errorf

---

### TestConfigValidation

**ID**: `UT-SHE-TestConfigValidation`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:464`

**Description**: TestConfigValidation tests configuration validation

**Test Steps**:
1. Call Run
2. Call NoError
3. Call Run
4. Call Error
5. Call Run
   ... and 5 more steps

---

### TestConfigValidation

**ID**: `UT-VIS-TestConfigValidation`

**Package**: `vision`

**File**: `internal/llm/vision/vision_test.go:531`

**Description**: TestConfigValidation tests configuration validation

**Test Steps**:
1. Call Run
2. Call Validate
3. Call Error
4. Call Errorf

---

### TestConfirmationCoordinator_BatchMode

**ID**: `UT-CON-TestConfirmationCoordinator_BatchMode`

**Package**: `confirmation`

**File**: `internal/tools/confirmation/confirmation_test.go:412`

**Description**: Test 12: ConfirmationCoordinator - Batch mode

**Test Steps**:
1. Call Confirm
2. Call Background
3. Call Fatalf
4. Call Error
5. Call Contains
   ... and 1 more steps

---

### TestConfirmationCoordinator_EndToEndAllow

**ID**: `UT-CON-TestConfirmationCoordinator_EndToEndAllow`

**Package**: `confirmation`

**File**: `internal/tools/confirmation/confirmation_test.go:373`

**Description**: Test 11: ConfirmationCoordinator - End to end with allow

**Test Steps**:
1. Call Confirm
2. Call Background
3. Call Fatalf
4. Call Error
5. Call Error

---

### TestConfirmationCoordinator_ResetChoices

**ID**: `UT-CON-TestConfirmationCoordinator_ResetChoices`

**Package**: `confirmation`

**File**: `internal/tools/confirmation/confirmation_test.go:488`

**Description**: Test 14: ConfirmationCoordinator - Reset choices

**Test Steps**:
1. Call SetUserChoice
2. Call GetUserChoice
3. Call Error
4. Call ResetChoices
5. Call GetUserChoice
   ... and 1 more steps

---

### TestConfirmationCoordinator_UserChoicePersistence

**ID**: `UT-CON-TestConfirmationCoordinator_UserChoicePersistence`

**Package**: `confirmation`

**File**: `internal/tools/confirmation/confirmation_test.go:441`

**Description**: Test 13: ConfirmationCoordinator - User choice persistence

**Test Steps**:
1. Call Confirm
2. Call Background
3. Call Fatalf
4. Call Errorf
5. Call Confirm
   ... and 5 more steps

---

### TestConflictResolver_DetectModification

**ID**: `UT-MUL-TestConflictResolver_DetectModification`

**Package**: `multiedit`

**File**: `internal/tools/multiedit/multiedit_test.go:156`

**Description**: Test 7: Conflict Detection

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call WriteFile
4. Call NoError
5. Call DetectConflicts
   ... and 3 more steps

---

### TestConsoleMonitor

**ID**: `UT-BRO-TestConsoleMonitor`

**Package**: `browser`

**File**: `internal/tools/browser/browser_test.go:334`

**Description**: TestConsoleMonitor tests console monitoring

**Test Steps**:
1. Call Run
2. Call NotNil
3. Call NotNil
4. Call GetMessages
5. Call NotNil
   ... and 5 more steps

---

### TestContentInspection

**ID**: `UT-VIS-TestContentInspection`

**Package**: `vision`

**File**: `internal/llm/vision/vision_test.go:223`

**Description**: TestContentInspection tests magic number-based content inspection

**Test Steps**:
1. Call Run
2. Call NewReader
3. Call InspectContent
4. Call Fatalf
5. Call Errorf
   ... and 1 more steps

---

### TestContextCancellation

**ID**: `UT-SHE-TestContextCancellation`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:295`

**Description**: TestContextCancellation tests context cancellation

**Test Steps**:
1. Call WithCancel
2. Call Background
3. Call Sleep
4. Call Execute
5. Call NoError
   ... and 1 more steps

---

### TestControllerIntegration

**ID**: `UT-AUT-TestControllerIntegration`

**Package**: `autonomy`

**File**: `internal/workflow/autonomy/autonomy_test.go:418`

**Description**: TestControllerIntegration tests full controller integration

**Test Steps**:
1. Call Fatalf
2. Call Background
3. Call SetMode
4. Call Fatalf
5. Call GetCurrentMode
   ... and 5 more steps

---

### TestConvertToCacheable

**ID**: `UT-LLM-TestConvertToCacheable`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:51`

**Description**: TestConvertToCacheable tests converting regular messages to cacheable messages

**Test Steps**:
1. Call Run
2. Call Len
3. Call Equal
4. Call Equal
5. Call Nil

---

### TestCoordinatorAgentStats

**ID**: `UT-AGE-TestCoordinatorAgentStats`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:825`

**Description**: TestCoordinatorAgentStats tests agent statistics

**Test Steps**:
1. Call RegisterAgent
2. Call GetAgentStats
3. Call Contains
4. Call Equal
5. Call Equal
   ... and 3 more steps

---

### TestCoordinatorAgentTaskCount

**ID**: `UT-AGE-TestCoordinatorAgentTaskCount`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:679`

**Description**: TestCoordinatorAgentTaskCount tests tracking agent task counts

**Test Steps**:
1. Call RegisterAgent
2. Call NewTask
3. Call TaskType
4. Call Background
5. Call SubmitTask
   ... and 5 more steps

---

### TestCoordinatorCircuitBreaker

**ID**: `UT-AGE-TestCoordinatorCircuitBreaker`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:661`

**Description**: TestCoordinatorCircuitBreaker tests circuit breaker state management

**Test Steps**:
1. Call RegisterAgent
2. Call GetCircuitBreakerState
3. Call ID
4. Call GetCircuitBreakerStats
5. Call NotNil

---

### TestCoordinatorConcurrentTaskExecution

**ID**: `UT-AGE-TestCoordinatorConcurrentTaskExecution`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:369`

**Description**: TestCoordinatorConcurrentTaskExecution tests concurrent task execution

**Test Steps**:
1. Call RegisterAgent
2. Call Background
3. Call NewTask
4. Call TaskType
5. Call SubmitTask
   ... and 5 more steps

---

### TestCoordinatorConcurrentTaskSubmission

**ID**: `UT-AGE-TestCoordinatorConcurrentTaskSubmission`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:335`

**Description**: TestCoordinatorConcurrentTaskSubmission tests concurrent task submission

**Test Steps**:
1. Call Background
2. Call NewTask
3. Call TaskType
4. Call SubmitTask
5. Call Equal

---

### TestCoordinatorContextCancellation

**ID**: `UT-AGE-TestCoordinatorContextCancellation`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:432`

**Description**: TestCoordinatorContextCancellation tests handling of context cancellation

**Test Steps**:
1. Call After
2. Call Now
3. Call Done
4. Call Err
5. Call RegisterAgent
   ... and 5 more steps

---

### TestCoordinatorEmptyAgentRegistry

**ID**: `UT-AGE-TestCoordinatorEmptyAgentRegistry`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:773`

**Description**: TestCoordinatorEmptyAgentRegistry tests behavior with no registered agents

**Test Steps**:
1. Call ListAgents
2. Call Len
3. Call GetAgentStats
4. Call Len
5. Call ListWorkflows
   ... and 1 more steps

---

### TestCoordinatorExecuteTask

**ID**: `UT-AGE-TestCoordinatorExecuteTask`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:97`

**Description**: TestCoordinatorExecuteTask tests task execution

**Test Steps**:
1. Call RegisterAgent
2. Call NewTask
3. Call TaskType
4. Call Background
5. Call SubmitTask
   ... and 5 more steps

---

### TestCoordinatorExecuteTaskAgentError

**ID**: `UT-AGE-TestCoordinatorExecuteTaskAgentError`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:164`

**Description**: TestCoordinatorExecuteTaskAgentError tests handling of agent execution errors

**Test Steps**:
1. Call New
2. Call RegisterAgent
3. Call NewTask
4. Call TaskType
5. Call Background
   ... and 5 more steps

---

### TestCoordinatorExecuteTaskNoAgent

**ID**: `UT-AGE-TestCoordinatorExecuteTaskNoAgent`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:139`

**Description**: TestCoordinatorExecuteTaskNoAgent tests execution when no suitable agent exists

**Test Steps**:
1. Call NewTask
2. Call TaskType
3. Call Background
4. Call SubmitTask
5. Call NoError
   ... and 5 more steps

---

### TestCoordinatorExecuteTaskNotFound

**ID**: `UT-AGE-TestCoordinatorExecuteTaskNotFound`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:127`

**Description**: TestCoordinatorExecuteTaskNotFound tests execution of non-existent task

**Test Steps**:
1. Call Background
2. Call ExecuteTask
3. Call Error
4. Call Nil
5. Call Contains
   ... and 1 more steps

---

### TestCoordinatorGetAgentStats

**ID**: `UT-AGE-TestCoordinatorGetAgentStats`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:295`

**Description**: TestCoordinatorGetAgentStats tests agent statistics retrieval

**Test Steps**:
1. Call RegisterAgent
2. Call RegisterAgent
3. Call GetAgentStats
4. Call NotNil
5. Call Contains
   ... and 4 more steps

---

### TestCoordinatorGetResult

**ID**: `UT-AGE-TestCoordinatorGetResult`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:227`

**Description**: TestCoordinatorGetResult tests result retrieval

**Test Steps**:
1. Call RegisterAgent
2. Call NewTask
3. Call TaskType
4. Call Background
5. Call SubmitTask
   ... and 5 more steps

---

### TestCoordinatorGetResultNotFound

**ID**: `UT-AGE-TestCoordinatorGetResultNotFound`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:258`

**Description**: TestCoordinatorGetResultNotFound tests result retrieval for non-existent task

**Test Steps**:
1. Call GetResult
2. Call Error
3. Call Nil
4. Call Contains
5. Call Error

---

### TestCoordinatorGetTaskStatus

**ID**: `UT-AGE-TestCoordinatorGetTaskStatus`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:194`

**Description**: TestCoordinatorGetTaskStatus tests task status retrieval

**Test Steps**:
1. Call NewTask
2. Call TaskType
3. Call Background
4. Call SubmitTask
5. Call NoError
   ... and 4 more steps

---

### TestCoordinatorGetTaskStatusNotFound

**ID**: `UT-AGE-TestCoordinatorGetTaskStatusNotFound`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:217`

**Description**: TestCoordinatorGetTaskStatusNotFound tests status retrieval for non-existent task

**Test Steps**:
1. Call GetTaskStatus
2. Call Error
3. Call Nil
4. Call Contains
5. Call Error

---

### TestCoordinatorListAgents

**ID**: `UT-AGE-TestCoordinatorListAgents`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:268`

**Description**: TestCoordinatorListAgents tests agent listing

**Test Steps**:
1. Call RegisterAgent
2. Call RegisterAgent
3. Call RegisterAgent
4. Call ListAgents
5. Call Len
   ... and 4 more steps

---

### TestCoordinatorListAgentsByCapability

**ID**: `UT-AGE-TestCoordinatorListAgentsByCapability`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:709`

**Description**: TestCoordinatorListAgentsByCapability tests finding agents by capability

**Test Steps**:
1. Call RegisterAgent
2. Call RegisterAgent
3. Call ListAgents
4. Call Len
5. Call ID
   ... and 3 more steps

---

### TestCoordinatorMultipleAgentsSameType

**ID**: `UT-AGE-TestCoordinatorMultipleAgentsSameType`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:571`

**Description**: TestCoordinatorMultipleAgentsSameType tests task execution with multiple agents of same type

**Test Steps**:
1. Call RegisterAgent
2. Call RegisterAgent
3. Call RegisterAgent
4. Call NewTask
5. Call TaskType
   ... and 5 more steps

---

### TestCoordinatorMultipleCapabilities

**ID**: `UT-AGE-TestCoordinatorMultipleCapabilities`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:790`

**Description**: TestCoordinatorMultipleCapabilities tests agent with multiple capabilities

**Test Steps**:
1. Call RegisterAgent
2. Call NewTask
3. Call TaskType
4. Call Background
5. Call SubmitTask
   ... and 4 more steps

---

### TestCoordinatorShutdown

**ID**: `UT-AGE-TestCoordinatorShutdown`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:321`

**Description**: TestCoordinatorShutdown tests coordinator shutdown

**Test Steps**:
1. Call RegisterAgent
2. Call Background
3. Call Shutdown
4. Call NoError

---

### TestCoordinatorSubmitNilTask

**ID**: `UT-AGE-TestCoordinatorSubmitNilTask`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:87`

**Description**: TestCoordinatorSubmitNilTask tests submitting nil task

**Test Steps**:
1. Call Background
2. Call SubmitTask
3. Call Error
4. Call Contains
5. Call Error

---

### TestCoordinatorSubmitTask

**ID**: `UT-AGE-TestCoordinatorSubmitTask`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:63`

**Description**: TestCoordinatorSubmitTask tests task submission

**Test Steps**:
1. Call NewTask
2. Call TaskType
3. Call Background
4. Call SubmitTask
5. Call NoError
   ... and 4 more steps

---

### TestCoordinatorTaskPriority

**ID**: `UT-AGE-TestCoordinatorTaskPriority`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:606`

**Description**: TestCoordinatorTaskPriority tests task priority handling

**Test Steps**:
1. Call RegisterAgent
2. Call Background
3. Call NewTask
4. Call TaskType
5. Call NewTask
   ... and 5 more steps

---

### TestCoordinatorTaskStatusTransitions

**ID**: `UT-AGE-TestCoordinatorTaskStatusTransitions`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:735`

**Description**: TestCoordinatorTaskStatusTransitions tests task status through lifecycle

**Test Steps**:
1. Call RegisterAgent
2. Call NewTask
3. Call TaskType
4. Call Background
5. Call SubmitTask
   ... and 5 more steps

---

### TestCoordinatorWorkflowList

**ID**: `UT-AGE-TestCoordinatorWorkflowList`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:845`

**Description**: TestCoordinatorWorkflowList tests workflow listing

**Test Steps**:
1. Call RegisterAgent
2. Call ListWorkflows
3. Call Len

---

### TestCountLines

**ID**: `UT-TYP-TestCountLines`

**Package**: `types`

**File**: `internal/agent/types/utils_test.go:10`

**Description**: TestCountLines tests the countLines utility function

**Test Steps**:
1. Call Run
2. Call Equal

---

### TestCountLines

**ID**: `UT-MAP-TestCountLines`

**Package**: `mapping`

**File**: `internal/tools/mapping/mapping_test.go:362`

**Description**: TestCountLines tests line counting

**Test Steps**:
1. Call Run
2. Call Equal

---

### TestCreateDefaultConfig

**ID**: `UT-CON-TestCreateDefaultConfig`

**Package**: `config`

**File**: `internal/config/config_test.go:155`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call NoError
4. Call Stat
5. Call NoError
   ... and 5 more steps

---

### TestCreateProject

**ID**: `UT-PRO-TestCreateProject`

**Package**: `project`

**File**: `internal/project/manager_test.go:20`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call NoError
3. Call RemoveAll
4. Call Join
5. Call WriteFile
   ... and 5 more steps

---

### TestCreateProject_InvalidPath

**ID**: `UT-PRO-TestCreateProject_InvalidPath`

**Package**: `project`

**File**: `internal/project/manager_test.go:42`

**Description**: No description provided

**Test Steps**:
1. Call CreateProject
2. Call Background
3. Call Error

---

### TestCreateSnapshot

**ID**: `UT-SNA-TestCreateSnapshot`

**Package**: `snapshots`

**File**: `internal/workflow/snapshots/snapshots_test.go:119`

**Description**: TestCreateSnapshot tests basic snapshot creation

**Test Steps**:
1. Call Fatalf
2. Call Background
3. Call CreateSnapshot
4. Call Fatalf
5. Call Error
   ... and 4 more steps

---

### TestCreateSnapshot_AutoGenerate

**ID**: `UT-SNA-TestCreateSnapshot_AutoGenerate`

**Package**: `snapshots`

**File**: `internal/workflow/snapshots/snapshots_test.go:189`

**Description**: TestCreateSnapshot_AutoGenerate tests auto-generated descriptions

**Test Steps**:
1. Call Fatalf
2. Call Background
3. Call CreateSnapshot
4. Call Fatalf
5. Call Error
   ... and 3 more steps

---

### TestCreateSnapshot_NoChanges

**ID**: `UT-SNA-TestCreateSnapshot_NoChanges`

**Package**: `snapshots`

**File**: `internal/workflow/snapshots/snapshots_test.go:165`

**Description**: TestCreateSnapshot_NoChanges tests snapshot creation with no changes

**Test Steps**:
1. Call Fatalf
2. Call Background
3. Call CreateSnapshot
4. Call Error
5. Call Contains
   ... and 2 more steps

---

### TestCreateTaskFromData

**ID**: `UT-TYP-TestCreateTaskFromData`

**Package**: `types`

**File**: `internal/agent/types/planning_agent_test.go:191`

**Description**: TestCreateTaskFromData tests task creation from parsed data

**Test Steps**:
1. Call NoError
2. Call Run
3. Call createTaskFromData
4. Call NotNil
5. Call Equal
   ... and 5 more steps

---

### TestCustomBudgetCreation

**ID**: `UT-LLM-TestCustomBudgetCreation`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:29`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call NotNil
3. Call Greater
4. Call Greater
5. Call Greater
   ... and 4 more steps

---

### TestDangerDetector_DetectDeleteOperation

**ID**: `UT-CON-TestDangerDetector_DetectDeleteOperation`

**Package**: `confirmation`

**File**: `internal/tools/confirmation/confirmation_test.go:131`

**Description**: Test 4: DangerDetector - Detect delete operation

**Test Steps**:
1. Call Detect
2. Call Errorf
3. Call Error
4. Call Error

---

### TestDangerDetector_DetectGitForcePush

**ID**: `UT-CON-TestDangerDetector_DetectGitForcePush`

**Package**: `confirmation`

**File**: `internal/tools/confirmation/confirmation_test.go:178`

**Description**: Test 6: DangerDetector - Detect git force push

**Test Steps**:
1. Call Detect
2. Call Errorf
3. Call Error

---

### TestDangerDetector_DetectSudoCommand

**ID**: `UT-CON-TestDangerDetector_DetectSudoCommand`

**Package**: `confirmation`

**File**: `internal/tools/confirmation/confirmation_test.go:200`

**Description**: Test 7: DangerDetector - Detect sudo command

**Test Steps**:
1. Call Detect
2. Call Errorf

---

### TestDangerDetector_DetectSystemFiles

**ID**: `UT-CON-TestDangerDetector_DetectSystemFiles`

**Package**: `confirmation`

**File**: `internal/tools/confirmation/confirmation_test.go:156`

**Description**: Test 5: DangerDetector - Detect system file operation

**Test Steps**:
1. Call Detect
2. Call Errorf
3. Call Error

---

### TestDangerousPatternDetection

**ID**: `UT-SHE-TestDangerousPatternDetection`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:161`

**Description**: TestDangerousPatternDetection tests dangerous pattern detection

**Test Steps**:
1. Call Run
2. Call Execute
3. Call Background
4. Call Error
5. Call ErrorAs

---

### TestDatabase_Close

**ID**: `UT-DAT-TestDatabase_Close`

**Package**: `database`

**File**: `internal/database/database_test.go:44`

**Description**: No description provided

**Test Steps**:
1. Call Close

---

### TestDatabase_GetDB

**ID**: `UT-DAT-TestDatabase_GetDB`

**Package**: `database`

**File**: `internal/database/database_test.go:51`

**Description**: No description provided

**Test Steps**:
1. Call GetDB
2. Call Error
3. Call Nil
4. Call Contains
5. Call Error

---

### TestDatabase_HealthCheck

**ID**: `UT-DAT-TestDatabase_HealthCheck`

**Package**: `database`

**File**: `internal/database/database_test.go:60`

**Description**: No description provided

**Test Steps**:
1. Call HealthCheck
2. Call Error
3. Call Contains
4. Call Error

---

### TestDebuggingAgentApplyFix

**ID**: `UT-TYP-TestDebuggingAgentApplyFix`

**Package**: `types`

**File**: `internal/agent/types/debugging_agent_test.go:548`

**Description**: TestDebuggingAgentApplyFix tests the applyFix helper function

**Test Steps**:
1. Call Run
2. Call NoError
3. Call Background
4. Call applyFix
5. Call NoError
   ... and 5 more steps

---

### TestDebuggingAgentCollaborate

**ID**: `UT-TYP-TestDebuggingAgentCollaborate`

**Package**: `types`

**File**: `internal/agent/types/debugging_agent_test.go:220`

**Description**: TestDebuggingAgentCollaborate tests collaboration with testing agents

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call NewBaseAgent
5. Call Background
   ... and 5 more steps

---

### TestDebuggingAgentDetermineDiagnosticCommands

**ID**: `UT-TYP-TestDebuggingAgentDetermineDiagnosticCommands`

**Package**: `types`

**File**: `internal/agent/types/debugging_agent_test.go:269`

**Description**: TestDebuggingAgentDetermineDiagnosticCommands tests diagnostic command generation

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Run
5. Call determineDiagnosticCommands
   ... and 5 more steps

---

### TestDebuggingAgentExecuteBasic

**ID**: `UT-TYP-TestDebuggingAgentExecuteBasic`

**Package**: `types`

**File**: `internal/agent/types/debugging_agent_test.go:108`

**Description**: TestDebuggingAgentExecuteBasic tests basic error analysis

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call NewTask
   ... and 5 more steps

---

### TestDebuggingAgentExecuteLLMError

**ID**: `UT-TYP-TestDebuggingAgentExecuteLLMError`

**Package**: `types`

**File**: `internal/agent/types/debugging_agent_test.go:184`

**Description**: TestDebuggingAgentExecuteLLMError tests LLM generation error

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call NewTask
   ... and 5 more steps

---

### TestDebuggingAgentExecuteMissingError

**ID**: `UT-TYP-TestDebuggingAgentExecuteMissingError`

**Package**: `types`

**File**: `internal/agent/types/debugging_agent_test.go:150`

**Description**: TestDebuggingAgentExecuteMissingError tests error when error message is missing

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call NewTask
   ... and 5 more steps

---

### TestDebuggingAgentGenerateFixedCode

**ID**: `UT-TYP-TestDebuggingAgentGenerateFixedCode`

**Package**: `types`

**File**: `internal/agent/types/debugging_agent_test.go:664`

**Description**: TestDebuggingAgentGenerateFixedCode tests the generateFixedCode helper function

**Test Steps**:
1. Call Run
2. Call NoError
3. Call Background
4. Call generateFixedCode
5. Call NoError
   ... and 5 more steps

---

### TestDebuggingAgentInitialize

**ID**: `UT-TYP-TestDebuggingAgentInitialize`

**Package**: `types`

**File**: `internal/agent/types/debugging_agent_test.go:66`

**Description**: TestDebuggingAgentInitialize tests agent initialization

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call Initialize
   ... and 3 more steps

---

### TestDebuggingAgentMetrics

**ID**: `UT-TYP-TestDebuggingAgentMetrics`

**Package**: `types`

**File**: `internal/agent/types/debugging_agent_test.go:296`

**Description**: TestDebuggingAgentMetrics tests metrics recording

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call NewTask
   ... and 5 more steps

---

### TestDebuggingAgentReadFile

**ID**: `UT-TYP-TestDebuggingAgentReadFile`

**Package**: `types`

**File**: `internal/agent/types/debugging_agent_test.go:335`

**Description**: TestDebuggingAgentReadFile tests the readFile helper function

**Test Steps**:
1. Call Run
2. Call NoError
3. Call Background
4. Call readFile
5. Call NoError
   ... and 5 more steps

---

### TestDebuggingAgentRunDiagnostics

**ID**: `UT-TYP-TestDebuggingAgentRunDiagnostics`

**Package**: `types`

**File**: `internal/agent/types/debugging_agent_test.go:432`

**Description**: TestDebuggingAgentRunDiagnostics tests the runDiagnostics helper function

**Test Steps**:
1. Call Run
2. Call NoError
3. Call Background
4. Call runDiagnostics
5. Call NoError
   ... and 5 more steps

---

### TestDebuggingAgentShutdown

**ID**: `UT-TYP-TestDebuggingAgentShutdown`

**Package**: `types`

**File**: `internal/agent/types/debugging_agent_test.go:87`

**Description**: TestDebuggingAgentShutdown tests agent shutdown

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call Shutdown
   ... and 3 more steps

---

### TestDefaultCacheConfig

**ID**: `UT-LLM-TestDefaultCacheConfig`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:11`

**Description**: TestDefaultCacheConfig tests the default cache configuration

**Test Steps**:
1. Call True
2. Call Equal
3. Call Equal
4. Call Equal

---

### TestDefaultConfig

**ID**: `UT-REP-TestDefaultConfig`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:40`

**Description**: No description provided

**Test Steps**:
1. Call Error
2. Call Error
3. Call Error
4. Call Errorf

---

### TestDefaultConfig

**ID**: `UT-AUT-TestDefaultConfig`

**Package**: `auth`

**File**: `internal/auth/auth_test.go:76`

**Description**: No description provided

**Test Steps**:
1. Call NotEmpty
2. Call Equal
3. Call Equal
4. Call Equal

---

### TestDefaultReasoningConfig

**ID**: `UT-LLM-TestDefaultReasoningConfig`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:13`

**Description**: Test ReasoningConfig creation and defaults

**Test Steps**:
1. Call NotNil
2. Call False
3. Call True
4. Call False
5. Call Equal
   ... and 3 more steps

---

### TestDefaultRetryConfig

**ID**: `UT-NOT-TestDefaultRetryConfig`

**Package**: `notification`

**File**: `internal/notification/retry_test.go:48`

**Description**: No description provided

**Test Steps**:
1. Call True
2. Call Equal
3. Call Equal
4. Call Equal
5. Call Equal

---

### TestDefaultRetryPolicy

**ID**: `UT-AGE-TestDefaultRetryPolicy`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:146`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call Equal
3. Call Equal
4. Call Equal
5. Call Equal

---

### TestDefaultTokenBudget

**ID**: `UT-LLM-TestDefaultTokenBudget`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:18`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call Equal
3. Call Equal
4. Call Equal
5. Call Equal
   ... and 1 more steps

---

### TestDefaultValidator

**ID**: `UT-EDI-TestDefaultValidator`

**Package**: `editor`

**File**: `internal/editor/editor_test.go:271`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Validate
3. Call Error
4. Call Errorf

---

### TestDefinitionGetSignature

**ID**: `UT-MAP-TestDefinitionGetSignature`

**Package**: `mapping`

**File**: `internal/tools/mapping/mapping_test.go:208`

**Description**: TestDefinitionGetSignature tests Definition.GetSignature

**Test Steps**:
1. Call Run
2. Call Equal
3. Call GetSignature

---

### TestDefinitionLineCount

**ID**: `UT-MAP-TestDefinitionLineCount`

**Package**: `mapping`

**File**: `internal/tools/mapping/mapping_test.go:256`

**Description**: TestDefinitionLineCount tests Definition.LineCount

**Test Steps**:
1. Call Equal
2. Call LineCount

---

### TestDefinitionType

**ID**: `UT-MAP-TestDefinitionType`

**Package**: `mapping`

**File**: `internal/tools/mapping/mapping_test.go:161`

**Description**: TestDefinitionType tests the DefinitionType enum

**Test Steps**:
1. Call Run
2. Call Equal
3. Call String

---

### TestDeleteProject

**ID**: `UT-PRO-TestDeleteProject`

**Package**: `project`

**File**: `internal/project/manager_test.go:155`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call NoError
3. Call RemoveAll
4. Call CreateProject
5. Call Background
   ... and 5 more steps

---

### TestDeleteSnapshot

**ID**: `UT-SNA-TestDeleteSnapshot`

**Package**: `snapshots`

**File**: `internal/workflow/snapshots/snapshots_test.go:409`

**Description**: TestDeleteSnapshot tests deleting a snapshot

**Test Steps**:
1. Call Fatalf
2. Call Background
3. Call CreateSnapshot
4. Call Fatalf
5. Call DeleteSnapshot
   ... and 3 more steps

---

### TestDependencyGraph

**ID**: `UT-MAP-TestDependencyGraph`

**Package**: `mapping`

**File**: `internal/tools/mapping/mapping_test.go:626`

**Description**: TestDependencyGraph tests dependency graph operations

**Test Steps**:
1. Call Run
2. Call GetDependencies
3. Call Equal
4. Call Run
5. Call GetDependents
   ... and 5 more steps

---

### TestDetectLanguage

**ID**: `UT-REP-TestDetectLanguage`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:62`

**Description**: Test language detection

**Test Steps**:
1. Call TempDir
2. Call detectLanguage
3. Call Errorf

---

### TestDetectLanguage

**ID**: `UT-GIT-TestDetectLanguage`

**Package**: `git`

**File**: `internal/tools/git/git_test.go:788`

**Description**: TestDetectLanguage tests language detection

**Test Steps**:
1. Call Run
2. Call Errorf

---

### TestDetectLanguage

**ID**: `UT-MAP-TestDetectLanguage`

**Package**: `mapping`

**File**: `internal/tools/mapping/mapping_test.go:265`

**Description**: TestDetectLanguage tests language detection

**Test Steps**:
1. Call Run
2. Call Equal

---

### TestDeviceManager_GetDefaultDevice

**ID**: `UT-VOI-TestDeviceManager_GetDefaultDevice`

**Package**: `voice`

**File**: `internal/tools/voice/voice_test.go:43`

**Description**: TestDeviceManager_GetDefaultDevice tests getting the default device

**Test Steps**:
1. Call Fatalf
2. Call GetDefaultDevice
3. Call Fatalf
4. Call Fatal
5. Call Error

---

### TestDeviceManager_ListDevices

**ID**: `UT-VOI-TestDeviceManager_ListDevices`

**Package**: `voice`

**File**: `internal/tools/voice/voice_test.go:13`

**Description**: TestDeviceManager_ListDevices tests device enumeration

**Test Steps**:
1. Call Fatalf
2. Call ListDevices
3. Call Background
4. Call Fatalf
5. Call Error
   ... and 3 more steps

---

### TestDeviceManager_SelectDevice

**ID**: `UT-VOI-TestDeviceManager_SelectDevice`

**Package**: `voice`

**File**: `internal/tools/voice/voice_test.go:64`

**Description**: TestDeviceManager_SelectDevice tests device selection

**Test Steps**:
1. Call Fatalf
2. Call ListDevices
3. Call Background
4. Call Skip
5. Call SelectDevice
   ... and 4 more steps

---

### TestDiffAnalyzer

**ID**: `UT-GIT-TestDiffAnalyzer`

**Package**: `git`

**File**: `internal/tools/git/git_test.go:465`

**Description**: TestDiffAnalyzer tests diff analysis

**Test Steps**:
1. Call Run
2. Call Analyze
3. Call Background
4. Call Fatalf
5. Call Errorf
   ... and 5 more steps

---

### TestDiffEditorApply

**ID**: `UT-EDI-TestDiffEditorApply`

**Package**: `editor`

**File**: `internal/editor/diff_editor_test.go:10`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Run
5. Call Join
   ... and 5 more steps

---

### TestDiffEditorApplyHunks

**ID**: `UT-EDI-TestDiffEditorApplyHunks`

**Package**: `editor`

**File**: `internal/editor/diff_editor_test.go:270`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call applyHunks
3. Call Error
4. Call Errorf
5. Call Errorf
   ... and 1 more steps

---

### TestDiffEditorLargeFile

**ID**: `UT-EDI-TestDiffEditorLargeFile`

**Package**: `editor`

**File**: `internal/editor/diff_editor_test.go:358`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Join
5. Call Repeat
   ... and 5 more steps

---

### TestDiffEditorNewFile

**ID**: `UT-EDI-TestDiffEditorNewFile`

**Package**: `editor`

**File**: `internal/editor/diff_editor_test.go:410`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Join
5. Call Apply
   ... and 4 more steps

---

### TestDiffEditorParseDiff

**ID**: `UT-EDI-TestDiffEditorParseDiff`

**Package**: `editor`

**File**: `internal/editor/diff_editor_test.go:133`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call parseDiff
3. Call Error
4. Call Errorf
5. Call Errorf

---

### TestDiffEditorParseHunkHeader

**ID**: `UT-EDI-TestDiffEditorParseHunkHeader`

**Package**: `editor`

**File**: `internal/editor/diff_editor_test.go:196`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call parseHunkHeader
3. Call Error
4. Call Errorf
5. Call Errorf
   ... and 3 more steps

---

### TestDiffManager_GenerateApply

**ID**: `UT-MUL-TestDiffManager_GenerateApply`

**Package**: `multiedit`

**File**: `internal/tools/multiedit/multiedit_test.go:107`

**Description**: Test 4: Diff Generation

**Test Steps**:
1. Call GenerateDiff
2. Call NoError
3. Call NotEmpty
4. Call Greater
5. Call ApplyDiff
   ... and 2 more steps

---

### TestDiffManager_ParseAndApply

**ID**: `UT-MUL-TestDiffManager_ParseAndApply`

**Package**: `multiedit`

**File**: `internal/tools/multiedit/multiedit_test.go:619`

**Description**: Test 19: Diff Parse and Apply

**Test Steps**:
1. Call GenerateDiff
2. Call NoError
3. Call ParseDiff
4. Call NoError
5. Call Greater
   ... and 3 more steps

---

### TestDiffManager_Stats

**ID**: `UT-MUL-TestDiffManager_Stats`

**Package**: `multiedit`

**File**: `internal/tools/multiedit/multiedit_test.go:126`

**Description**: Test 5: Diff Stats

**Test Steps**:
1. Call GenerateDiff
2. Call NoError
3. Call Greater
4. Call GreaterOrEqual

---

### TestDiscordChannel_GetConfig

**ID**: `UT-NOT-TestDiscordChannel_GetConfig`

**Package**: `notification`

**File**: `internal/notification/discord_test.go:267`

**Description**: No description provided

**Test Steps**:
1. Call GetConfig
2. Call NotNil
3. Call Equal

---

### TestDiscordChannel_GetName

**ID**: `UT-NOT-TestDiscordChannel_GetName`

**Package**: `notification`

**File**: `internal/notification/discord_test.go:276`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call GetName

---

### TestDiscordChannel_IsEnabled

**ID**: `UT-NOT-TestDiscordChannel_IsEnabled`

**Package**: `notification`

**File**: `internal/notification/discord_test.go:281`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Equal
3. Call IsEnabled

---

### TestDiscordChannel_Send

**ID**: `UT-NOT-TestDiscordChannel_Send`

**Package**: `notification`

**File**: `internal/notification/discord_test.go:44`

**Description**: No description provided

**Test Steps**:
1. Call Contains
2. Call Contains
3. Call Contains
4. Call Contains
5. Call Contains
   ... and 5 more steps

---

### TestDiscordChannel_Send_AllNotificationTypes

**ID**: `UT-NOT-TestDiscordChannel_Send_AllNotificationTypes`

**Package**: `notification`

**File**: `internal/notification/discord_test.go:205`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call WriteHeader
4. Call Close
5. Call Run
   ... and 3 more steps

---

### TestDiscordChannel_Send_ConcurrentRequests

**ID**: `UT-NOT-TestDiscordChannel_Send_ConcurrentRequests`

**Package**: `notification`

**File**: `internal/notification/discord_test.go:307`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call WriteHeader
4. Call Close
5. Call Send
   ... and 3 more steps

---

### TestDiscordChannel_Send_ContextCancellation

**ID**: `UT-NOT-TestDiscordChannel_Send_ContextCancellation`

**Package**: `notification`

**File**: `internal/notification/discord_test.go:341`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call WriteHeader
4. Call Close
5. Call WithCancel
   ... and 2 more steps

---

### TestDiscordChannel_Send_Disabled

**ID**: `UT-NOT-TestDiscordChannel_Send_Disabled`

**Package**: `notification`

**File**: `internal/notification/discord_test.go:176`

**Description**: No description provided

**Test Steps**:
1. Call Send
2. Call Background
3. Call Error
4. Call Contains
5. Call Error

---

### TestDiscordChannel_Send_LargePayload

**ID**: `UT-NOT-TestDiscordChannel_Send_LargePayload`

**Package**: `notification`

**File**: `internal/notification/discord_test.go:369`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call WriteHeader
4. Call Close
5. Call Send
   ... and 2 more steps

---

### TestDiscordChannel_Send_NetworkError

**ID**: `UT-NOT-TestDiscordChannel_Send_NetworkError`

**Package**: `notification`

**File**: `internal/notification/discord_test.go:190`

**Description**: No description provided

**Test Steps**:
1. Call Send
2. Call Background
3. Call Error
4. Call Contains
5. Call Error

---

### TestDiscordChannel_Send_WithMetadata

**ID**: `UT-NOT-TestDiscordChannel_Send_WithMetadata`

**Package**: `notification`

**File**: `internal/notification/discord_test.go:235`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Read
4. Call WriteHeader
5. Call Close
   ... and 5 more steps

---

### TestDiscoverFiles

**ID**: `UT-REP-TestDiscoverFiles`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:99`

**Description**: Test file discovery

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call MkdirAll
4. Call discoverFiles
5. Call Fatalf
   ... and 3 more steps

---

### TestDiscoverFilesIgnoresCommonDirs

**ID**: `UT-REP-TestDiscoverFilesIgnoresCommonDirs`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:133`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call MkdirAll
4. Call discoverFiles
5. Call Fatalf
   ... and 1 more steps

---

### TestDiskCacheManager

**ID**: `UT-MAP-TestDiskCacheManager`

**Package**: `mapping`

**File**: `internal/tools/mapping/mapping_test.go:439`

**Description**: TestDiskCacheManager tests cache operations

**Test Steps**:
1. Call TempDir
2. Call Run
3. Call Save
4. Call NoError
5. Call Load
   ... and 5 more steps

---

### TestDiskCache_Persistence

**ID**: `UT-WEB-TestDiskCache_Persistence`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:507`

**Description**: Test 21: Disk cache persistence

**Test Steps**:
1. Call TempDir
2. Call Set
3. Call Close
4. Call Get
5. Call True
   ... and 2 more steps

---

### TestDistributedWorkerManager

**ID**: `UT-WOR-TestDistributedWorkerManager`

**Package**: `worker`

**File**: `internal/worker/distributed_manager_test.go:10`

**Description**: TestDistributedWorkerManager tests the distributed worker manager

**Test Steps**:
1. Call Skip
2. Call Background
3. Call Initialize
4. Call Log
5. Call GetWorkerStats
   ... and 2 more steps

---

### TestEdgeCases_ConcurrentAccess

**ID**: `UT-LLM-TestEdgeCases_ConcurrentAccess`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:1071`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call String
3. Call New
4. Call CheckBudget
5. Call TrackRequest
   ... and 4 more steps

---

### TestEdgeCases_EmptySessionID

**ID**: `UT-LLM-TestEdgeCases_EmptySessionID`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:1114`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call CheckBudget
3. Call NoError
4. Call TrackRequest
5. Call GetSessionUsage
   ... and 2 more steps

---

### TestEdgeCases_NegativeValues

**ID**: `UT-LLM-TestEdgeCases_NegativeValues`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:1024`

**Description**: No description provided

**Test Steps**:
1. Call TrackRequest
2. Call GetSessionUsage
3. Call NoError
4. Call Equal
5. Call Background
   ... and 2 more steps

---

### TestEdgeCases_VeryLargeValues

**ID**: `UT-LLM-TestEdgeCases_VeryLargeValues`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:1045`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call CheckBudget
3. Call NoError
4. Call TrackRequest
5. Call GetSessionUsage
   ... and 3 more steps

---

### TestEdgeCases_ZeroBudgetValues

**ID**: `UT-LLM-TestEdgeCases_ZeroBudgetValues`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:1007`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call CheckBudget
3. Call Error

---

### TestEmailChannel_ExtractRecipients

**ID**: `UT-NOT-TestEmailChannel_ExtractRecipients`

**Package**: `notification`

**File**: `internal/notification/email_test.go:68`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call extractRecipients
3. Call Error
4. Call Contains
5. Call Error
   ... and 2 more steps

---

### TestEmailChannel_GetConfig

**ID**: `UT-NOT-TestEmailChannel_GetConfig`

**Package**: `notification`

**File**: `internal/notification/email_test.go:206`

**Description**: No description provided

**Test Steps**:
1. Call GetConfig
2. Call Equal
3. Call Equal
4. Call Equal
5. Call Equal
   ... and 1 more steps

---

### TestEmailChannel_Send_Disabled

**ID**: `UT-NOT-TestEmailChannel_Send_Disabled`

**Package**: `notification`

**File**: `internal/notification/email_test.go:190`

**Description**: No description provided

**Test Steps**:
1. Call Send
2. Call Background
3. Call Error
4. Call Contains
5. Call Error

---

### TestEntraTokenProvider_Caching

**ID**: `UT-LLM-TestEntraTokenProvider_Caching`

**Package**: `llm`

**File**: `internal/llm/azure_provider_test.go:439`

**Description**: Test 12: Entra token provider caching

**Test Steps**:
1. Call Add
2. Call Now
3. Call GetToken
4. Call Background
5. Call NoError
   ... and 5 more steps

---

### TestEnvSanitization

**ID**: `UT-SHE-TestEnvSanitization`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:403`

**Description**: TestEnvSanitization tests environment variable sanitization

**Test Steps**:
1. Call Run
2. Call Equal
3. Call NotContains
4. Call NotContains

---

### TestErrorRecovery

**ID**: `UT-PLA-TestErrorRecovery`

**Package**: `planmode`

**File**: `internal/workflow/planmode/planmode_test.go:724`

**Description**: Test error recovery

**Test Steps**:
1. Call Run
2. Call Execute
3. Call Background
4. Call NoError
5. Call NotNil
   ... and 4 more steps

---

### TestEscalation

**ID**: `UT-AUT-TestEscalation`

**Package**: `autonomy`

**File**: `internal/workflow/autonomy/autonomy_test.go:344`

**Description**: TestEscalation tests mode escalation

**Test Steps**:
1. Call Skip
2. Call Fatalf
3. Call Background
4. Call SetMode
5. Call Fatalf
   ... and 5 more steps

---

### TestEstimateCost

**ID**: `UT-LLM-TestEstimateCost`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:979`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call InDelta

---

### TestEstimateDuration

**ID**: `UT-TYP-TestEstimateDuration`

**Package**: `types`

**File**: `internal/agent/types/planning_agent_test.go:135`

**Description**: TestEstimateDuration tests the duration estimation method

**Test Steps**:
1. Call NoError
2. Call Run
3. Call estimateDuration
4. Call Equal

---

### TestEstimateDurationEdgeCases

**ID**: `UT-TYP-TestEstimateDurationEdgeCases`

**Package**: `types`

**File**: `internal/agent/types/planning_agent_test.go:286`

**Description**: TestEstimateDurationEdgeCases tests edge cases for duration estimation

**Test Steps**:
1. Call NoError
2. Call Run
3. Call estimateDuration
4. Call Equal
5. Call Duration
   ... and 5 more steps

---

### TestEstimateTokens

**ID**: `UT-LLM-TestEstimateTokens`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:597`

**Description**: Test helper functions

**Test Steps**:
1. Call Repeat
2. Call Repeat
3. Call Run
4. Call Equal

---

### TestEstimateTokens_MessagesOnly

**ID**: `UT-LLM-TestEstimateTokens_MessagesOnly`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:910`

**Description**: No description provided

**Test Steps**:
1. Call Repeat
2. Call Run
3. Call Equal

---

### TestEstimateTokens_WithTools

**ID**: `UT-LLM-TestEstimateTokens_WithTools`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:959`

**Description**: No description provided

**Test Steps**:
1. Call Greater

---

### TestEvaluatePolicy

**ID**: `UT-COM-TestEvaluatePolicy`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:480`

**Description**: Test 17: Evaluate Policy

**Test Steps**:
1. Call Greater
2. Call GreaterOrEqual
3. Call Equal

---

### TestEventBus_AllEventTypes

**ID**: `UT-EVE-TestEventBus_AllEventTypes`

**Package**: `event`

**File**: `internal/event/bus_test.go:431`

**Description**: No description provided

**Test Steps**:
1. Call Lock
2. Call Unlock
3. Call Subscribe
4. Call Publish
5. Call Background
   ... and 1 more steps

---

### TestEventBus_AsyncMode

**ID**: `UT-EVE-TestEventBus_AsyncMode`

**Package**: `event`

**File**: `internal/event/bus_test.go:153`

**Description**: No description provided

**Test Steps**:
1. Call Add
2. Call Sleep
3. Call Done
4. Call Subscribe
5. Call Now
   ... and 5 more steps

---

### TestEventBus_AsyncModeErrors

**ID**: `UT-EVE-TestEventBus_AsyncModeErrors`

**Package**: `event`

**File**: `internal/event/bus_test.go:246`

**Description**: No description provided

**Test Steps**:
1. Call Add
2. Call Done
3. Call New
4. Call Subscribe
5. Call Publish
   ... and 5 more steps

---

### TestEventBus_ClearErrors

**ID**: `UT-EVE-TestEventBus_ClearErrors`

**Package**: `event`

**File**: `internal/event/bus_test.go:354`

**Description**: No description provided

**Test Steps**:
1. Call New
2. Call Subscribe
3. Call Publish
4. Call Background
5. Call GetErrors
   ... and 4 more steps

---

### TestEventBus_ConcurrentPublish

**ID**: `UT-EVE-TestEventBus_ConcurrentPublish`

**Package**: `event`

**File**: `internal/event/bus_test.go:379`

**Description**: No description provided

**Test Steps**:
1. Call AddInt32
2. Call Subscribe
3. Call Add
4. Call Done
5. Call Publish
   ... and 4 more steps

---

### TestEventBus_ConcurrentSubscribe

**ID**: `UT-EVE-TestEventBus_ConcurrentSubscribe`

**Package**: `event`

**File**: `internal/event/bus_test.go:410`

**Description**: No description provided

**Test Steps**:
1. Call Add
2. Call Done
3. Call Subscribe
4. Call Wait
5. Call Equal
   ... and 1 more steps

---

### TestEventBus_ErrorLogLimit

**ID**: `UT-EVE-TestEventBus_ErrorLogLimit`

**Package**: `event`

**File**: `internal/event/bus_test.go:471`

**Description**: No description provided

**Test Steps**:
1. Call New
2. Call Subscribe
3. Call Publish
4. Call Background
5. Call GetErrors
   ... and 1 more steps

---

### TestEventBus_EventData

**ID**: `UT-EVE-TestEventBus_EventData`

**Package**: `event`

**File**: `internal/event/bus_test.go:301`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call Publish
3. Call Background
4. Call NoError
5. Call Equal
   ... and 4 more steps

---

### TestEventBus_EventIDAndTimestamp

**ID**: `UT-EVE-TestEventBus_EventIDAndTimestamp`

**Package**: `event`

**File**: `internal/event/bus_test.go:275`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call Publish
3. Call Background
4. Call NoError
5. Call NotEmpty
   ... and 2 more steps

---

### TestEventBus_GetSubscribedEvents

**ID**: `UT-EVE-TestEventBus_GetSubscribedEvents`

**Package**: `event`

**File**: `internal/event/bus_test.go:336`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call Subscribe
3. Call Subscribe
4. Call GetSubscribedEvents
5. Call Equal
   ... and 3 more steps

---

### TestEventBus_MultipleHandlers

**ID**: `UT-EVE-TestEventBus_MultipleHandlers`

**Package**: `event`

**File**: `internal/event/bus_test.go:115`

**Description**: No description provided

**Test Steps**:
1. Call AddInt32
2. Call AddInt32
3. Call Subscribe
4. Call Subscribe
5. Call Publish
   ... and 4 more steps

---

### TestEventBus_NoSubscribers

**ID**: `UT-EVE-TestEventBus_NoSubscribers`

**Package**: `event`

**File**: `internal/event/bus_test.go:141`

**Description**: No description provided

**Test Steps**:
1. Call Publish
2. Call Background
3. Call NoError

---

### TestEventBus_PublishAndWait

**ID**: `UT-EVE-TestEventBus_PublishAndWait`

**Package**: `event`

**File**: `internal/event/bus_test.go:187`

**Description**: No description provided

**Test Steps**:
1. Call Sleep
2. Call Subscribe
3. Call Now
4. Call PublishAndWait
5. Call Background
   ... and 4 more steps

---

### TestEventBus_Subscribe

**ID**: `UT-EVE-TestEventBus_Subscribe`

**Package**: `event`

**File**: `internal/event/bus_test.go:43`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call Equal
3. Call GetSubscriberCount
4. Call Equal
5. Call GetTotalSubscribers
   ... and 4 more steps

---

### TestEventBus_SubscribeMultiple

**ID**: `UT-EVE-TestEventBus_SubscribeMultiple`

**Package**: `event`

**File**: `internal/event/bus_test.go:69`

**Description**: No description provided

**Test Steps**:
1. Call AddInt32
2. Call SubscribeMultiple
3. Call Equal
4. Call GetTotalSubscribers
5. Call Publish
   ... and 3 more steps

---

### TestEventBus_SyncModeErrors

**ID**: `UT-EVE-TestEventBus_SyncModeErrors`

**Package**: `event`

**File**: `internal/event/bus_test.go:214`

**Description**: No description provided

**Test Steps**:
1. Call New
2. Call New
3. Call Subscribe
4. Call Subscribe
5. Call Subscribe
   ... and 5 more steps

---

### TestEventBus_Unsubscribe

**ID**: `UT-EVE-TestEventBus_Unsubscribe`

**Package**: `event`

**File**: `internal/event/bus_test.go:101`

**Description**: No description provided

**Test Steps**:
1. Call Subscribe
2. Call Equal
3. Call GetSubscriberCount
4. Call Unsubscribe
5. Call Equal
   ... and 1 more steps

---

### TestEventNotificationHandler_EndToEnd

**ID**: `UT-NOT-TestEventNotificationHandler_EndToEnd`

**Package**: `notification`

**File**: `internal/notification/event_handler_test.go:357`

**Description**: No description provided

**Test Steps**:
1. Call NewEventBus
2. Call RegisterWithEventBus
3. Call RegisterChannel
4. Call AddRule
5. Call Now
   ... and 5 more steps

---

### TestEventNotificationHandler_HandleEvent_SystemError

**ID**: `UT-NOT-TestEventNotificationHandler_HandleEvent_SystemError`

**Package**: `notification`

**File**: `internal/notification/event_handler_test.go:241`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call AddRule
3. Call HandleEvent
4. Call Background
5. Call NoError
   ... and 5 more steps

---

### TestEventNotificationHandler_HandleEvent_SystemStartup

**ID**: `UT-NOT-TestEventNotificationHandler_HandleEvent_SystemStartup`

**Package**: `notification`

**File**: `internal/notification/event_handler_test.go:283`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call AddRule
3. Call HandleEvent
4. Call Background
5. Call NoError
   ... and 4 more steps

---

### TestEventNotificationHandler_HandleEvent_TaskCompleted

**ID**: `UT-NOT-TestEventNotificationHandler_HandleEvent_TaskCompleted`

**Package**: `notification`

**File**: `internal/notification/event_handler_test.go:21`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call AddRule
3. Call HandleEvent
4. Call Background
5. Call NoError
   ... and 5 more steps

---

### TestEventNotificationHandler_HandleEvent_TaskFailed

**ID**: `UT-NOT-TestEventNotificationHandler_HandleEvent_TaskFailed`

**Package**: `notification`

**File**: `internal/notification/event_handler_test.go:71`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call AddRule
3. Call HandleEvent
4. Call Background
5. Call NoError
   ... and 5 more steps

---

### TestEventNotificationHandler_HandleEvent_TaskStartedIgnored

**ID**: `UT-NOT-TestEventNotificationHandler_HandleEvent_TaskStartedIgnored`

**Package**: `notification`

**File**: `internal/notification/event_handler_test.go:322`

**Description**: No description provided

**Test Steps**:
1. Call HandleEvent
2. Call Background
3. Call NoError

---

### TestEventNotificationHandler_HandleEvent_WorkerDisconnected

**ID**: `UT-NOT-TestEventNotificationHandler_HandleEvent_WorkerDisconnected`

**Package**: `notification`

**File**: `internal/notification/event_handler_test.go:197`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call AddRule
3. Call HandleEvent
4. Call Background
5. Call NoError
   ... and 5 more steps

---

### TestEventNotificationHandler_HandleEvent_WorkflowCompleted

**ID**: `UT-NOT-TestEventNotificationHandler_HandleEvent_WorkflowCompleted`

**Package**: `notification`

**File**: `internal/notification/event_handler_test.go:114`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call AddRule
3. Call HandleEvent
4. Call Background
5. Call NoError
   ... and 5 more steps

---

### TestEventNotificationHandler_HandleEvent_WorkflowFailed

**ID**: `UT-NOT-TestEventNotificationHandler_HandleEvent_WorkflowFailed`

**Package**: `notification`

**File**: `internal/notification/event_handler_test.go:155`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call AddRule
3. Call HandleEvent
4. Call Background
5. Call NoError
   ... and 5 more steps

---

### TestEventNotificationHandler_RegisterWithEventBus

**ID**: `UT-NOT-TestEventNotificationHandler_RegisterWithEventBus`

**Package**: `notification`

**File**: `internal/notification/event_handler_test.go:341`

**Description**: No description provided

**Test Steps**:
1. Call NewEventBus
2. Call RegisterWithEventBus
3. Call Greater
4. Call GetSubscriberCount
5. Call Greater
   ... and 5 more steps

---

### TestExecuteBuildingWorkflow

**ID**: `UT-WOR-TestExecuteBuildingWorkflow`

**Package**: `workflow`

**File**: `internal/workflow/executor_test.go:41`

**Description**: No description provided

**Test Steps**:
1. Call NewManager
2. Call MkdirTemp
3. Call NoError
4. Call RemoveAll
5. Call CreateProject
   ... and 5 more steps

---

### TestExecutePlanningWorkflow

**ID**: `UT-WOR-TestExecutePlanningWorkflow`

**Package**: `workflow`

**File**: `internal/workflow/executor_test.go:20`

**Description**: No description provided

**Test Steps**:
1. Call NewManager
2. Call MkdirTemp
3. Call NoError
4. Call RemoveAll
5. Call CreateProject
   ... and 5 more steps

---

### TestExecuteRefactoringWorkflow

**ID**: `UT-WOR-TestExecuteRefactoringWorkflow`

**Package**: `workflow`

**File**: `internal/workflow/executor_test.go:79`

**Description**: No description provided

**Test Steps**:
1. Call NewManager
2. Call MkdirTemp
3. Call NoError
4. Call RemoveAll
5. Call CreateProject
   ... and 5 more steps

---

### TestExecuteStep_Analysis

**ID**: `UT-WOR-TestExecuteStep_Analysis`

**Package**: `workflow`

**File**: `internal/workflow/executor_test.go:106`

**Description**: No description provided

**Test Steps**:
1. Call NewManager
2. Call MkdirTemp
3. Call NoError
4. Call RemoveAll
5. Call CreateProject
   ... and 5 more steps

---

### TestExecuteStep_Generation

**ID**: `UT-WOR-TestExecuteStep_Generation`

**Package**: `workflow`

**File**: `internal/workflow/executor_test.go:130`

**Description**: No description provided

**Test Steps**:
1. Call NewManager
2. Call MkdirTemp
3. Call NoError
4. Call RemoveAll
5. Call CreateProject
   ... and 5 more steps

---

### TestExecuteStep_UnknownAction

**ID**: `UT-WOR-TestExecuteStep_UnknownAction`

**Package**: `workflow`

**File**: `internal/workflow/executor_test.go:154`

**Description**: No description provided

**Test Steps**:
1. Call NewManager
2. Call MkdirTemp
3. Call NoError
4. Call RemoveAll
5. Call CreateProject
   ... and 5 more steps

---

### TestExecuteTestingWorkflow

**ID**: `UT-WOR-TestExecuteTestingWorkflow`

**Package**: `workflow`

**File**: `internal/workflow/executor_test.go:60`

**Description**: No description provided

**Test Steps**:
1. Call NewManager
2. Call MkdirTemp
3. Call NoError
4. Call RemoveAll
5. Call CreateProject
   ... and 5 more steps

---

### TestExecuteWorkflow_InvalidProject

**ID**: `UT-WOR-TestExecuteWorkflow_InvalidProject`

**Package**: `workflow`

**File**: `internal/workflow/executor_test.go:98`

**Description**: No description provided

**Test Steps**:
1. Call NewManager
2. Call ExecutePlanningWorkflow
3. Call Background
4. Call Error

---

### TestExecutionStatus

**ID**: `UT-SHE-TestExecutionStatus`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:500`

**Description**: TestExecutionStatus tests execution status tracking

**Test Steps**:
1. Call ExecuteAsync
2. Call Background
3. Call NoError
4. Call Sleep
5. Call GetStatus
   ... and 5 more steps

---

### TestExecutor

**ID**: `UT-PLA-TestExecutor`

**Package**: `planmode`

**File**: `internal/workflow/planmode/planmode_test.go:430`

**Description**: Test Executor

**Test Steps**:
1. Call Run
2. Call ExecuteStep
3. Call Background
4. Call NoError
5. Call True
   ... and 5 more steps

---

### TestExtractFileSymbols

**ID**: `UT-REP-TestExtractFileSymbols`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:163`

**Description**: Test symbol extraction

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call extractFileSymbols
4. Call Fatalf
5. Call Errorf
   ... and 2 more steps

---

### TestExtractFunctionName

**ID**: `UT-GIT-TestExtractFunctionName`

**Package**: `git`

**File**: `internal/tools/git/git_test.go:738`

**Description**: TestExtractFunctionName tests function name extraction

**Test Steps**:
1. Call Run
2. Call Errorf

---

### TestExtractReasoningTrace_CustomTags

**ID**: `UT-LLM-TestExtractReasoningTrace_CustomTags`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:126`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call Len
3. Call Equal
4. Call Equal

---

### TestExtractReasoningTrace_DisabledConfig

**ID**: `UT-LLM-TestExtractReasoningTrace_DisabledConfig`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:69`

**Description**: Test ExtractReasoningTrace

**Test Steps**:
1. Call NotNil
2. Call Empty
3. Call Equal
4. Call Equal
5. Call Greater

---

### TestExtractReasoningTrace_MultipleTags

**ID**: `UT-LLM-TestExtractReasoningTrace_MultipleTags`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:140`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call Len
3. Call Equal
4. Call Equal
5. Call Equal
   ... and 2 more steps

---

### TestExtractReasoningTrace_MultipleThinkingBlocks

**ID**: `UT-LLM-TestExtractReasoningTrace_MultipleThinkingBlocks`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:97`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call Len
3. Call Equal
4. Call Equal
5. Call Contains
   ... and 1 more steps

---

### TestExtractReasoningTrace_NestedTags

**ID**: `UT-LLM-TestExtractReasoningTrace_NestedTags`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:114`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call Len
3. Call Contains
4. Call Equal

---

### TestExtractReasoningTrace_NoThinkingBlocks

**ID**: `UT-LLM-TestExtractReasoningTrace_NoThinkingBlocks`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:159`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call Empty
3. Call Equal
4. Call Equal
5. Call Greater

---

### TestExtractReasoningTrace_SingleThinkingBlock

**ID**: `UT-LLM-TestExtractReasoningTrace_SingleThinkingBlock`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:82`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call Len
3. Call Equal
4. Call Equal
5. Call Greater
   ... and 2 more steps

---

### TestExtractRelativeIndentation

**ID**: `UT-MAP-TestExtractRelativeIndentation`

**Package**: `mapping`

**File**: `internal/tools/mapping/mapping_test.go:660`

**Description**: TestExtractRelativeIndentation tests relative indentation extraction

**Test Steps**:
1. Call NotEmpty
2. Call Split
3. Call True
4. Call HasPrefix

---

### TestExtractSymbols

**ID**: `UT-REP-TestExtractSymbols`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:477`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call ParseFile
4. Call ExtractSymbols
5. Call Fatalf
   ... and 2 more steps

---

### TestFetcher_FetchMultiple

**ID**: `UT-WEB-TestFetcher_FetchMultiple`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:420`

**Description**: Test 18: Multiple concurrent fetches

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Sleep
4. Call WriteHeader
5. Call Write
   ... and 5 more steps

---

### TestFetcher_Fetch_NotFound

**ID**: `UT-WEB-TestFetcher_Fetch_NotFound`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:70`

**Description**: Test 4: HTTP fetch - 404 error

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call WriteHeader
4. Call Write
5. Call Close
   ... and 5 more steps

---

### TestFetcher_Fetch_Redirect

**ID**: `UT-WEB-TestFetcher_Fetch_Redirect`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:91`

**Description**: Test 5: HTTP fetch - redirect handling

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call WriteHeader
4. Call Write
5. Call Close
   ... and 5 more steps

---

### TestFetcher_Fetch_Success

**ID**: `UT-WEB-TestFetcher_Fetch_Success`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:47`

**Description**: Test 3: HTTP fetch with mock server - success

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Set
4. Call Header
5. Call WriteHeader
   ... and 5 more steps

---

### TestFetcher_Fetch_Timeout

**ID**: `UT-WEB-TestFetcher_Fetch_Timeout`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:118`

**Description**: Test 6: HTTP fetch - timeout

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Sleep
4. Call Write
5. Call Close
   ... and 5 more steps

---

### TestFetcher_ValidateURL_Invalid

**ID**: `UT-WEB-TestFetcher_ValidateURL_Invalid`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:303`

**Description**: Test 14: URL validation - invalid URLs

**Test Steps**:
1. Call NoError
2. Call Close
3. Call validateURL
4. Call Error

---

### TestFetcher_ValidateURL_Valid

**ID**: `UT-WEB-TestFetcher_ValidateURL_Valid`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:283`

**Description**: Test 13: URL validation - valid URLs

**Test Steps**:
1. Call NoError
2. Call Close
3. Call validateURL
4. Call NoError

---

### TestFileEditor

**ID**: `UT-FIL-TestFileEditor`

**Package**: `filesystem`

**File**: `internal/tools/filesystem/filesystem_test.go:375`

**Description**: TestFileEditor tests file editing operations

**Test Steps**:
1. Call MkdirTemp
2. Call Fatal
3. Call RemoveAll
4. Call Fatal
5. Call Background
   ... and 5 more steps

---

### TestFileReader

**ID**: `UT-FIL-TestFileReader`

**Package**: `filesystem`

**File**: `internal/tools/filesystem/filesystem_test.go:55`

**Description**: TestFileReader tests file reading operations

**Test Steps**:
1. Call MkdirTemp
2. Call Fatal
3. Call RemoveAll
4. Call Fatal
5. Call Background
   ... and 5 more steps

---

### TestFileSearcher

**ID**: `UT-FIL-TestFileSearcher`

**Package**: `filesystem`

**File**: `internal/tools/filesystem/filesystem_test.go:505`

**Description**: TestFileSearcher tests file searching operations

**Test Steps**:
1. Call MkdirTemp
2. Call Fatal
3. Call RemoveAll
4. Call Join
5. Call MkdirAll
   ... and 5 more steps

---

### TestFileWriter

**ID**: `UT-FIL-TestFileWriter`

**Package**: `filesystem`

**File**: `internal/tools/filesystem/filesystem_test.go:174`

**Description**: TestFileWriter tests file writing operations

**Test Steps**:
1. Call MkdirTemp
2. Call Fatal
3. Call RemoveAll
4. Call Fatal
5. Call Background
   ... and 5 more steps

---

### TestFindBestVisionModel

**ID**: `UT-VIS-TestFindBestVisionModel`

**Package**: `vision`

**File**: `internal/llm/vision/vision_test.go:588`

**Description**: TestFindBestVisionModel tests finding the best vision model

**Test Steps**:
1. Call Background
2. Call Run
3. Call FindBestVisionModel
4. Call Fatalf
5. Call Fatal
   ... and 2 more steps

---

### TestFindConfigFile

**ID**: `UT-CON-TestFindConfigFile`

**Package**: `config`

**File**: `internal/config/config_test.go:140`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call WriteFile
4. Call NoError
5. Call Getenv
   ... and 3 more steps

---

### TestFindSuitableAgentBusyAgent

**ID**: `UT-AGE-TestFindSuitableAgentBusyAgent`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:537`

**Description**: TestFindSuitableAgentBusyAgent tests that busy agents are not selected

**Test Steps**:
1. Call SetStatus
2. Call RegisterAgent
3. Call RegisterAgent
4. Call NewTask
5. Call TaskType
   ... and 5 more steps

---

### TestFindSuitableAgentByCapability

**ID**: `UT-AGE-TestFindSuitableAgentByCapability`

**Package**: `agent`

**File**: `internal/agent/coordinator_test.go:505`

**Description**: TestFindSuitableAgentByCapability tests agent selection by capability

**Test Steps**:
1. Call RegisterAgent
2. Call RegisterAgent
3. Call NewTask
4. Call TaskType
5. Call Background
   ... and 5 more steps

---

### TestFormatReasoningPrompt

**ID**: `UT-LLM-TestFormatReasoningPrompt`

**Package**: `llm`

**File**: `internal/llm/provider_features_test.go:425`

**Description**: TestFormatReasoningPrompt tests reasoning prompt formatting

**Test Steps**:
1. Call Run
2. Call Contains
3. Call NotContains

---

### TestFormatReasoningPrompt_Claude

**ID**: `UT-LLM-TestFormatReasoningPrompt_Claude`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:292`

**Description**: No description provided

**Test Steps**:
1. Call Contains
2. Call Contains

---

### TestFormatReasoningPrompt_DeepSeek

**ID**: `UT-LLM-TestFormatReasoningPrompt_DeepSeek`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:302`

**Description**: No description provided

**Test Steps**:
1. Call Contains
2. Call Contains

---

### TestFormatReasoningPrompt_DisabledConfig

**ID**: `UT-LLM-TestFormatReasoningPrompt_DisabledConfig`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:273`

**Description**: Test FormatReasoningPrompt

**Test Steps**:
1. Call Equal

---

### TestFormatReasoningPrompt_OpenAI_O1

**ID**: `UT-LLM-TestFormatReasoningPrompt_OpenAI_O1`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:282`

**Description**: No description provided

**Test Steps**:
1. Call Contains

---

### TestFormatReasoningPrompt_WithEffortLevels

**ID**: `UT-LLM-TestFormatReasoningPrompt_WithEffortLevels`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:312`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Contains

---

### TestGeminiProvider_Close

**ID**: `UT-LLM-TestGeminiProvider_Close`

**Package**: `llm`

**File**: `internal/llm/gemini_provider_test.go:609`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call Close
3. Call NoError

---

### TestGeminiProvider_ErrorHandling

**ID**: `UT-LLM-TestGeminiProvider_ErrorHandling`

**Package**: `llm`

**File**: `internal/llm/gemini_provider_test.go:484`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call NewServer
3. Call HandlerFunc
4. Call WriteHeader
5. Call Write
   ... and 5 more steps

---

### TestGeminiProvider_Generate

**ID**: `UT-LLM-TestGeminiProvider_Generate`

**Package**: `llm`

**File**: `internal/llm/gemini_provider_test.go:215`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Contains
4. Call Contains
5. Call Get
   ... and 5 more steps

---

### TestGeminiProvider_GenerateWithSystemInstruction

**ID**: `UT-LLM-TestGeminiProvider_GenerateWithSystemInstruction`

**Package**: `llm`

**File**: `internal/llm/gemini_provider_test.go:287`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Decode
4. Call NewDecoder
5. Call NoError
   ... and 5 more steps

---

### TestGeminiProvider_GenerateWithTools

**ID**: `UT-LLM-TestGeminiProvider_GenerateWithTools`

**Package**: `llm`

**File**: `internal/llm/gemini_provider_test.go:345`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Decode
4. Call NewDecoder
5. Call NoError
   ... and 5 more steps

---

### TestGeminiProvider_GetCapabilities

**ID**: `UT-LLM-TestGeminiProvider_GetCapabilities`

**Package**: `llm`

**File**: `internal/llm/gemini_provider_test.go:163`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call GetCapabilities
3. Call NotEmpty
4. Call True
5. Call True
   ... and 3 more steps

---

### TestGeminiProvider_GetHealth

**ID**: `UT-LLM-TestGeminiProvider_GetHealth`

**Package**: `llm`

**File**: `internal/llm/gemini_provider_test.go:570`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Encode
4. Call NewEncoder
5. Call Close
   ... and 5 more steps

---

### TestGeminiProvider_GetModels

**ID**: `UT-LLM-TestGeminiProvider_GetModels`

**Package**: `llm`

**File**: `internal/llm/gemini_provider_test.go:128`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call GetModels
3. Call NotEmpty
4. Call Equal
5. Call Greater
   ... and 5 more steps

---

### TestGeminiProvider_GetName

**ID**: `UT-LLM-TestGeminiProvider_GetName`

**Package**: `llm`

**File**: `internal/llm/gemini_provider_test.go:117`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call Equal
3. Call GetName

---

### TestGeminiProvider_GetType

**ID**: `UT-LLM-TestGeminiProvider_GetType`

**Package**: `llm`

**File**: `internal/llm/gemini_provider_test.go:106`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call Equal
3. Call GetType

---

### TestGeminiProvider_IsAvailable

**ID**: `UT-LLM-TestGeminiProvider_IsAvailable`

**Package**: `llm`

**File**: `internal/llm/gemini_provider_test.go:188`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call NoError
3. Call IsAvailable
4. Call Background
5. Call Equal

---

### TestGeminiProvider_MassiveContext

**ID**: `UT-LLM-TestGeminiProvider_MassiveContext`

**Package**: `llm`

**File**: `internal/llm/gemini_provider_test.go:649`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call GetModels
3. Call NotEmpty
4. Call Contains
5. Call Contains

---

### TestGeminiProvider_MessageConversion

**ID**: `UT-LLM-TestGeminiProvider_MessageConversion`

**Package**: `llm`

**File**: `internal/llm/gemini_provider_test.go:621`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call convertMessages
3. Call Equal
4. Call Len
5. Call Equal
   ... and 2 more steps

---

### TestGeminiProvider_SafetySettings

**ID**: `UT-LLM-TestGeminiProvider_SafetySettings`

**Package**: `llm`

**File**: `internal/llm/gemini_provider_test.go:434`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Decode
4. Call NewDecoder
5. Call NoError
   ... and 5 more steps

---

### TestGenerateAgentID

**ID**: `UT-AGE-TestGenerateAgentID`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:628`

**Description**: No description provided

**Test Steps**:
1. Call Error
2. Call Error
3. Call Error

---

### TestGenerateDiff

**ID**: `UT-SNA-TestGenerateDiff`

**Package**: `snapshots`

**File**: `internal/workflow/snapshots/snapshots_test.go:516`

**Description**: TestGenerateDiff tests diff generation

**Test Steps**:
1. Call Fatalf
2. Call Background
3. Call CreateSnapshot
4. Call Fatalf
5. Call Command
   ... and 5 more steps

---

### TestGenerateMessage

**ID**: `UT-GIT-TestGenerateMessage`

**Package**: `git`

**File**: `internal/tools/git/git_test.go:283`

**Description**: TestGenerateMessage tests message generation without committing

**Test Steps**:
1. Call Join
2. Call WriteFile
3. Call Fatal
4. Call Command
5. Call Run
   ... and 5 more steps

---

### TestGenerateThemeFiles

**ID**: `UT-LOG-TestGenerateThemeFiles`

**Package**: `logo`

**File**: `internal/logo/processor_test.go:72`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call NoError
3. Call RemoveAll
4. Call Join
5. Call MkdirAll
   ... and 5 more steps

---

### TestGenerateWorkflowID

**ID**: `UT-AGE-TestGenerateWorkflowID`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:246`

**Description**: No description provided

**Test Steps**:
1. Call Sleep
2. Call NotEmpty
3. Call NotEmpty
4. Call NotEqual
5. Call Contains
   ... and 1 more steps

---

### TestGetActiveProject

**ID**: `UT-PRO-TestGetActiveProject`

**Package**: `project`

**File**: `internal/project/manager_test.go:111`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call NoError
3. Call RemoveAll
4. Call CreateProject
5. Call Background
   ... and 5 more steps

---

### TestGetActiveProject_NoActive

**ID**: `UT-PRO-TestGetActiveProject_NoActive`

**Package**: `project`

**File**: `internal/project/manager_test.go:128`

**Description**: No description provided

**Test Steps**:
1. Call GetActiveProject
2. Call Background
3. Call Error

---

### TestGetBudgetStatus_DailyStats

**ID**: `UT-LLM-TestGetBudgetStatus_DailyStats`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:746`

**Description**: No description provided

**Test Steps**:
1. Call TrackRequest
2. Call TrackRequest
3. Call GetBudgetStatus
4. Call Equal
5. Call Equal
   ... and 1 more steps

---

### TestGetBudgetStatus_NewSession

**ID**: `UT-LLM-TestGetBudgetStatus_NewSession`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:630`

**Description**: No description provided

**Test Steps**:
1. Call GetBudgetStatus
2. Call NotNil
3. Call Equal
4. Call Equal
5. Call Equal
   ... and 5 more steps

---

### TestGetBudgetStatus_PercentageCalculations

**ID**: `UT-LLM-TestGetBudgetStatus_PercentageCalculations`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:676`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call TrackRequest
3. Call GetBudgetStatus
4. Call InDelta
5. Call InDelta

---

### TestGetBudgetStatus_RemainingBudget

**ID**: `UT-LLM-TestGetBudgetStatus_RemainingBudget`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:724`

**Description**: No description provided

**Test Steps**:
1. Call TrackRequest
2. Call GetBudgetStatus
3. Call Equal
4. Call InDelta

---

### TestGetBudgetStatus_WithUsage

**ID**: `UT-LLM-TestGetBudgetStatus_WithUsage`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:649`

**Description**: No description provided

**Test Steps**:
1. Call TrackRequest
2. Call GetBudgetStatus
3. Call Equal
4. Call Equal
5. Call InDelta
   ... and 4 more steps

---

### TestGetDailyUsage_MultipleSessions

**ID**: `UT-LLM-TestGetDailyUsage_MultipleSessions`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:587`

**Description**: No description provided

**Test Steps**:
1. Call Format
2. Call Now
3. Call TrackRequest
4. Call TrackRequest
5. Call TrackRequest
   ... and 5 more steps

---

### TestGetDailyUsage_NonExistentDate

**ID**: `UT-LLM-TestGetDailyUsage_NonExistentDate`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:617`

**Description**: No description provided

**Test Steps**:
1. Call GetDailyUsage
2. Call Error
3. Call Contains
4. Call Error

---

### TestGetDailyUsage_SingleDay

**ID**: `UT-LLM-TestGetDailyUsage_SingleDay`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:569`

**Description**: No description provided

**Test Steps**:
1. Call Format
2. Call Now
3. Call TrackRequest
4. Call GetDailyUsage
5. Call NoError
   ... and 5 more steps

---

### TestGetDefaultRateLimiter

**ID**: `UT-NOT-TestGetDefaultRateLimiter`

**Package**: `notification`

**File**: `internal/notification/ratelimit_test.go:284`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Equal
3. Call Equal

---

### TestGetEnvIntOrDefault

**ID**: `UT-CON-TestGetEnvIntOrDefault`

**Package**: `config`

**File**: `internal/config/config_test.go:185`

**Description**: No description provided

**Test Steps**:
1. Call Setenv
2. Call Unsetenv
3. Call Equal
4. Call Equal
5. Call Setenv
   ... and 2 more steps

---

### TestGetEnvOrDefault

**ID**: `UT-CON-TestGetEnvOrDefault`

**Package**: `config`

**File**: `internal/config/config_test.go:174`

**Description**: No description provided

**Test Steps**:
1. Call Setenv
2. Call Unsetenv
3. Call Equal
4. Call Equal

---

### TestGetFormatComplexity

**ID**: `UT-EDI-TestGetFormatComplexity`

**Package**: `editor`

**File**: `internal/editor/model_formats_test.go:183`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Errorf

---

### TestGetLanguageQueries

**ID**: `UT-REP-TestGetLanguageQueries`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:882`

**Description**: No description provided

**Test Steps**:
1. Call Errorf
2. Call Error

---

### TestGetModelCapability

**ID**: `UT-EDI-TestGetModelCapability`

**Package**: `editor`

**File**: `internal/editor/model_formats_test.go:54`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Errorf
3. Call Errorf
4. Call Errorf
5. Call Errorf

---

### TestGetOptimalContext

**ID**: `UT-REP-TestGetOptimalContext`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:218`

**Description**: Test GetOptimalContext

**Test Steps**:
1. Call TempDir
2. Call GetOptimalContext
3. Call Fatalf
4. Call Error
5. Call Contains
   ... and 1 more steps

---

### TestGetOptimalContextTokenBudget

**ID**: `UT-REP-TestGetOptimalContextTokenBudget`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:286`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Repeat
3. Call Join
4. Call GetOptimalContext
5. Call Fatalf
   ... and 1 more steps

---

### TestGetOptimalContextWithChangedFiles

**ID**: `UT-REP-TestGetOptimalContextWithChangedFiles`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:263`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call GetOptimalContext
4. Call Fatalf
5. Call Error

---

### TestGetProject

**ID**: `UT-PRO-TestGetProject`

**Package**: `project`

**File**: `internal/project/manager_test.go:49`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call NoError
3. Call RemoveAll
4. Call CreateProject
5. Call Background
   ... and 5 more steps

---

### TestGetProject_NotFound

**ID**: `UT-PRO-TestGetProject_NotFound`

**Package**: `project`

**File**: `internal/project/manager_test.go:64`

**Description**: No description provided

**Test Steps**:
1. Call GetProject
2. Call Background
3. Call Error

---

### TestGetReasoningBudgetRecommendation

**ID**: `UT-LLM-TestGetReasoningBudgetRecommendation`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:474`

**Description**: Test GetReasoningBudgetRecommendation

**Test Steps**:
1. Call Run
2. Call Equal

---

### TestGetSnapshot

**ID**: `UT-SNA-TestGetSnapshot`

**Package**: `snapshots`

**File**: `internal/workflow/snapshots/snapshots_test.go:223`

**Description**: TestGetSnapshot tests retrieving a snapshot

**Test Steps**:
1. Call Fatalf
2. Call Background
3. Call CreateSnapshot
4. Call Fatalf
5. Call GetSnapshot
   ... and 3 more steps

---

### TestGetSnapshotFiles

**ID**: `UT-SNA-TestGetSnapshotFiles`

**Package**: `snapshots`

**File**: `internal/workflow/snapshots/snapshots_test.go:709`

**Description**: TestGetSnapshotFiles tests getting file list from snapshot

**Test Steps**:
1. Call Fatalf
2. Call Background
3. Call CreateSnapshot
4. Call Fatalf
5. Call GetSnapshotFiles
   ... and 2 more steps

---

### TestGetSnapshot_NotFound

**ID**: `UT-SNA-TestGetSnapshot_NotFound`

**Package**: `snapshots`

**File**: `internal/workflow/snapshots/snapshots_test.go:259`

**Description**: TestGetSnapshot_NotFound tests retrieving non-existent snapshot

**Test Steps**:
1. Call Fatalf
2. Call Background
3. Call GetSnapshot
4. Call Error

---

### TestGetStatistics

**ID**: `UT-REP-TestGetStatistics`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:403`

**Description**: Test statistics

**Test Steps**:
1. Call TempDir
2. Call GetStatistics
3. Call Fatalf
4. Call Errorf
5. Call Error
   ... and 1 more steps

---

### TestGetTestDirectory

**ID**: `UT-TYP-TestGetTestDirectory`

**Package**: `types`

**File**: `internal/agent/types/testing_agent_test.go:418`

**Description**: TestGetTestDirectory tests test directory extraction

**Test Steps**:
1. Call Run
2. Call Equal

---

### TestGetTestFilePath

**ID**: `UT-TYP-TestGetTestFilePath`

**Package**: `types`

**File**: `internal/agent/types/testing_agent_test.go:381`

**Description**: TestGetTestFilePath tests test file path generation

**Test Steps**:
1. Call Run
2. Call Equal

---

### TestGitPolicy_WarnForcePush

**ID**: `UT-CON-TestGitPolicy_WarnForcePush`

**Package**: `confirmation`

**File**: `internal/tools/confirmation/confirmation_test.go:603`

**Description**: Test 19: GitPolicy - Warn force push

**Test Steps**:
1. Call SetPolicy
2. Call Fatalf
3. Call Evaluate
4. Call Fatalf
5. Call Errorf

---

### TestGrayToASCII

**ID**: `UT-LOG-TestGrayToASCII`

**Package**: `logo`

**File**: `internal/logo/processor_test.go:37`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call Equal
3. Call Equal

---

### TestGroqProvider_Close

**ID**: `UT-LLM-TestGroqProvider_Close`

**Package**: `llm`

**File**: `internal/llm/groq_provider_test.go:605`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call Close
3. Call NoError

---

### TestGroqProvider_ErrorHandling

**ID**: `UT-LLM-TestGroqProvider_ErrorHandling`

**Package**: `llm`

**File**: `internal/llm/groq_provider_test.go:411`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call NewServer
3. Call HandlerFunc
4. Call WriteHeader
5. Call Write
   ... and 5 more steps

---

### TestGroqProvider_Generate

**ID**: `UT-LLM-TestGroqProvider_Generate`

**Package**: `llm`

**File**: `internal/llm/groq_provider_test.go:198`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Equal
4. Call Contains
5. Call Get
   ... and 5 more steps

---

### TestGroqProvider_GenerateStream

**ID**: `UT-LLM-TestGroqProvider_GenerateStream`

**Package**: `llm`

**File**: `internal/llm/groq_provider_test.go:285`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Equal
4. Call Equal
5. Call Get
   ... and 5 more steps

---

### TestGroqProvider_GetCapabilities

**ID**: `UT-LLM-TestGroqProvider_GetCapabilities`

**Package**: `llm`

**File**: `internal/llm/groq_provider_test.go:144`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call GetCapabilities
3. Call NotEmpty
4. Call True
5. Call True
   ... and 5 more steps

---

### TestGroqProvider_GetHealth

**ID**: `UT-LLM-TestGroqProvider_GetHealth`

**Package**: `llm`

**File**: `internal/llm/groq_provider_test.go:524`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Unix
4. Call Now
5. Call Encode
   ... and 5 more steps

---

### TestGroqProvider_GetModels

**ID**: `UT-LLM-TestGroqProvider_GetModels`

**Package**: `llm`

**File**: `internal/llm/groq_provider_test.go:114`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call GetModels
3. Call NotEmpty
4. Call GreaterOrEqual
5. Call Equal
   ... and 5 more steps

---

### TestGroqProvider_GetName

**ID**: `UT-LLM-TestGroqProvider_GetName`

**Package**: `llm`

**File**: `internal/llm/groq_provider_test.go:103`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call Equal
3. Call GetName

---

### TestGroqProvider_GetType

**ID**: `UT-LLM-TestGroqProvider_GetType`

**Package**: `llm`

**File**: `internal/llm/groq_provider_test.go:92`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call Equal
3. Call GetType

---

### TestGroqProvider_HTTP2Support

**ID**: `UT-LLM-TestGroqProvider_HTTP2Support`

**Package**: `llm`

**File**: `internal/llm/groq_provider_test.go:665`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call True
3. Call Equal
4. Call Equal

---

### TestGroqProvider_HealthCheckFailure

**ID**: `UT-LLM-TestGroqProvider_HealthCheckFailure`

**Package**: `llm`

**File**: `internal/llm/groq_provider_test.go:582`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call WriteHeader
4. Call Write
5. Call Close
   ... and 5 more steps

---

### TestGroqProvider_IsAvailable

**ID**: `UT-LLM-TestGroqProvider_IsAvailable`

**Package**: `llm`

**File**: `internal/llm/groq_provider_test.go:171`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call NoError
3. Call IsAvailable
4. Call Background
5. Call Equal

---

### TestGroqProvider_LatencyMetricsRetrieval

**ID**: `UT-LLM-TestGroqProvider_LatencyMetricsRetrieval`

**Package**: `llm`

**File**: `internal/llm/groq_provider_test.go:389`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call GetLatencyMetrics
3. Call NotNil
4. Call Equal
5. Call RecordRequest
   ... and 3 more steps

---

### TestGroqProvider_LatencyTracking

**ID**: `UT-LLM-TestGroqProvider_LatencyTracking`

**Package**: `llm`

**File**: `internal/llm/groq_provider_test.go:370`

**Description**: No description provided

**Test Steps**:
1. Call RecordRequest
2. Call RecordRequest
3. Call RecordRequest
4. Call GetMetrics
5. Call Equal
   ... and 5 more steps

---

### TestGroqProvider_ModelContextSizes

**ID**: `UT-LLM-TestGroqProvider_ModelContextSizes`

**Package**: `llm`

**File**: `internal/llm/groq_provider_test.go:617`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call GetModels
3. Call HasPrefix
4. Call Equal
5. Call Equal
   ... and 2 more steps

---

### TestGroqProvider_PercentileCalculations

**ID**: `UT-LLM-TestGroqProvider_PercentileCalculations`

**Package**: `llm`

**File**: `internal/llm/groq_provider_test.go:641`

**Description**: No description provided

**Test Steps**:
1. Call Greater
2. Call Duration
3. Call Greater
4. Call Greater
5. Call LessOrEqual

---

### TestGuardrails

**ID**: `UT-AUT-TestGuardrails`

**Package**: `autonomy`

**File**: `internal/workflow/autonomy/autonomy_test.go:273`

**Description**: TestGuardrails tests safety guardrails

**Test Steps**:
1. Call Background
2. Call Run
3. Call Check
4. Call Fatalf
5. Call Errorf
   ... and 1 more steps

---

### TestHardwareDetectionErrorHandling

**ID**: `UT-HAR-TestHardwareDetectionErrorHandling`

**Package**: `hardware`

**File**: `internal/hardware/detector_test.go:60`

**Description**: TestHardwareDetectionErrorHandling tests error handling in hardware detection

**Test Steps**:
1. Call CanRunModel
2. Call Logf
3. Call GetCompilationFlags
4. Call GetCompilationFlags
5. Call Error
   ... and 1 more steps

---

### TestHardwareDetector

**ID**: `UT-HAR-TestHardwareDetector`

**Package**: `hardware`

**File**: `internal/hardware/detector_test.go:8`

**Description**: TestHardwareDetector tests the hardware detector

**Test Steps**:
1. Call Detect
2. Call Fatalf
3. Call Error
4. Call Error
5. Call Error
   ... and 5 more steps

---

### TestHealthCheckEndpoint

**ID**: `UT-SER-TestHealthCheckEndpoint`

**Package**: `server`

**File**: `internal/server/server_test.go:126`

**Description**: No description provided

**Test Steps**:
1. Call New
2. Call setupRoutes
3. Call NewRecorder
4. Call NewRequest
5. Call ServeHTTP
   ... and 1 more steps

---

### TestHelperFunctions

**ID**: `UT-BRO-TestHelperFunctions`

**Package**: `browser`

**File**: `internal/tools/browser/browser_test.go:465`

**Description**: TestHelperFunctions tests helper functions

**Test Steps**:
1. Call Run
2. Call Equal
3. Call String
4. Call Equal
5. Call String
   ... and 5 more steps

---

### TestHybridStrategy_Execute

**ID**: `UT-COM-TestHybridStrategy_Execute`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:328`

**Description**: Test 8: Hybrid Strategy

**Test Steps**:
1. Call Execute
2. Call Background
3. Call NoError
4. Call Less
5. Call Greater
   ... and 1 more steps

---

### TestImageDetectionBase64

**ID**: `UT-VIS-TestImageDetectionBase64`

**Package**: `vision`

**File**: `internal/llm/vision/vision_test.go:129`

**Description**: TestImageDetectionBase64 tests base64 image detection

**Test Steps**:
1. Call Background
2. Call Run
3. Call Detect
4. Call Fatalf
5. Call Errorf

---

### TestImageDetectionByExtension

**ID**: `UT-VIS-TestImageDetectionByExtension`

**Package**: `vision`

**File**: `internal/llm/vision/vision_test.go:90`

**Description**: TestImageDetectionByExtension tests image detection using file extensions

**Test Steps**:
1. Call Run
2. Call DetectInFile
3. Call Fatalf
4. Call Errorf

---

### TestImageDetectionByMIME

**ID**: `UT-VIS-TestImageDetectionByMIME`

**Package**: `vision`

**File**: `internal/llm/vision/vision_test.go:12`

**Description**: TestImageDetectionByMIME tests image detection using MIME types

**Test Steps**:
1. Call Background
2. Call Run
3. Call Detect
4. Call Fatalf
5. Call Errorf
   ... and 1 more steps

---

### TestImageDetectionURL

**ID**: `UT-VIS-TestImageDetectionURL`

**Package**: `vision`

**File**: `internal/llm/vision/vision_test.go:177`

**Description**: TestImageDetectionURL tests URL-based image detection

**Test Steps**:
1. Call Run
2. Call DetectInText
3. Call Fatalf
4. Call Errorf

---

### TestImportAnalyzer

**ID**: `UT-MAP-TestImportAnalyzer`

**Package**: `mapping`

**File**: `internal/tools/mapping/mapping_test.go:754`

**Description**: TestImportAnalyzer tests import analysis

**Test Steps**:
1. Call Run
2. Call ResolveDependencies
3. Call NotNil
4. Call IsType
5. Call Run
   ... and 4 more steps

---

### TestIntegrationFileSystemTools

**ID**: `UT-TOO-TestIntegrationFileSystemTools`

**Package**: `tools`

**File**: `internal/tools/registry_test.go:77`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call Fatalf
4. Call Close
5. Call Background
   ... and 5 more steps

---

### TestIntegrationFullCachingWorkflow

**ID**: `UT-LLM-TestIntegrationFullCachingWorkflow`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:777`

**Description**: TestIntegrationFullCachingWorkflow tests a complete caching workflow

**Test Steps**:
1. Call NotNil
2. Call Equal
3. Call Greater
4. Call Greater
5. Call UpdateMetrics
   ... and 5 more steps

---

### TestIntegrationMultiEdit

**ID**: `UT-TOO-TestIntegrationMultiEdit`

**Package**: `tools`

**File**: `internal/tools/registry_test.go:193`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call Join
4. Call WriteFile
5. Call WriteFile
   ... and 5 more steps

---

### TestIntegrationNotebook

**ID**: `UT-TOO-TestIntegrationNotebook`

**Package**: `tools`

**File**: `internal/tools/registry_test.go:286`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call Fatalf
4. Call Fatalf
5. Call Close
   ... and 5 more steps

---

### TestIntegrationRealMessageSequence

**ID**: `UT-LLM-TestIntegrationRealMessageSequence`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:835`

**Description**: TestIntegrationRealMessageSequence tests with a realistic conversation

**Test Steps**:
1. Call Run
2. Call NotNil
3. Call Nil

---

### TestIntegrationShellTools

**ID**: `UT-TOO-TestIntegrationShellTools`

**Package**: `tools`

**File**: `internal/tools/registry_test.go:151`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Fatalf
3. Call Close
4. Call Background
5. Call Run
   ... and 5 more steps

---

### TestIntegrationTaskTracker

**ID**: `UT-TOO-TestIntegrationTaskTracker`

**Package**: `tools`

**File**: `internal/tools/registry_test.go:230`

**Description**: No description provided

**Test Steps**:
1. Call Fatalf
2. Call Close
3. Call Background
4. Call Run
5. Call Execute
   ... and 5 more steps

---

### TestIntegrationToolCachingScenarios

**ID**: `UT-LLM-TestIntegrationToolCachingScenarios`

**Package**: `llm`

**File**: `internal/llm/cache_control_test.go:908`

**Description**: TestIntegrationToolCachingScenarios tests various tool caching scenarios

**Test Steps**:
1. Call Run
2. Call NotNil
3. Call Equal
4. Call Nil

---

### TestIntegration_FullWorkflow

**ID**: `UT-LLM-TestIntegration_FullWorkflow`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:1136`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call CheckBudget
3. Call NoError
4. Call TrackRequest
5. Call GetSessionUsage
   ... and 5 more steps

---

### TestIntegration_MultiSession

**ID**: `UT-LLM-TestIntegration_MultiSession`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:1217`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call CheckBudget
3. Call NoError
4. Call TrackRequest
5. Call CheckBudget
   ... and 5 more steps

---

### TestIntegration_ResetAndContinue

**ID**: `UT-LLM-TestIntegration_ResetAndContinue`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:1275`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call TrackRequest
3. Call GetSessionUsage
4. Call NoError
5. Call Equal
   ... and 5 more steps

---

### TestIntegration_SessionCleanup

**ID**: `UT-LLM-TestIntegration_SessionCleanup`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:1311`

**Description**: No description provided

**Test Steps**:
1. Call String
2. Call New
3. Call TrackRequest
4. Call GetAllSessionsUsage
5. Call Len
   ... and 5 more steps

---

### TestInvalidWorkDir

**ID**: `UT-SHE-TestInvalidWorkDir`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:332`

**Description**: TestInvalidWorkDir tests invalid working directory

**Test Steps**:
1. Call Execute
2. Call Background
3. Call Error

---

### TestInvalidateFile

**ID**: `UT-REP-TestInvalidateFile`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:345`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call extractFileSymbols
4. Call Sleep
5. Call InvalidateFile
   ... and 2 more steps

---

### TestIsReasoningModel_Claude

**ID**: `UT-LLM-TestIsReasoningModel_Claude`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:359`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Equal
3. Call Equal

---

### TestIsReasoningModel_DeepSeek

**ID**: `UT-LLM-TestIsReasoningModel_DeepSeek`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:383`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Equal
3. Call Equal

---

### TestIsReasoningModel_OpenAI

**ID**: `UT-LLM-TestIsReasoningModel_OpenAI`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:334`

**Description**: Test IsReasoningModel

**Test Steps**:
1. Call Run
2. Call Equal
3. Call Equal

---

### TestIsReasoningModel_QwQ

**ID**: `UT-LLM-TestIsReasoningModel_QwQ`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:406`

**Description**: No description provided

**Test Steps**:
1. Call True
2. Call Equal

---

### TestIsRetryableError

**ID**: `UT-NOT-TestIsRetryableError`

**Package**: `notification`

**File**: `internal/notification/retry_test.go:82`

**Description**: No description provided

**Test Steps**:
1. Call New
2. Call New
3. Call New
4. Call New
5. Call New
   ... and 2 more steps

---

### TestIsSupported

**ID**: `UT-MAP-TestIsSupported`

**Package**: `mapping`

**File**: `internal/tools/mapping/mapping_test.go:292`

**Description**: TestIsSupported tests language support check

**Test Steps**:
1. Call Run
2. Call Equal

---

### TestIterationLimits

**ID**: `UT-AUT-TestIterationLimits`

**Package**: `autonomy`

**File**: `internal/workflow/autonomy/autonomy_test.go:471`

**Description**: TestIterationLimits tests iteration limits per mode

**Test Steps**:
1. Call Background
2. Call Run
3. Call DisableRule
4. Call Check
5. Call Fatalf
   ... and 1 more steps

---

### TestLanguageRegistry

**ID**: `UT-MAP-TestLanguageRegistry`

**Package**: `mapping`

**File**: `internal/tools/mapping/mapping_test.go:313`

**Description**: TestLanguageRegistry tests the language registry

**Test Steps**:
1. Call Run
2. Call List
3. Call NotNil
4. Call Run
5. Call GetLanguageInfo
   ... and 5 more steps

---

### TestLevelMonitor

**ID**: `UT-VOI-TestLevelMonitor`

**Package**: `voice`

**File**: `internal/tools/voice/voice_test.go:214`

**Description**: TestLevelMonitor tests audio level monitoring

**Test Steps**:
1. Call Update
2. Call GetLevels
3. Call Errorf
4. Call Errorf
5. Call IsZero
   ... and 4 more steps

---

### TestLineEditorApply

**ID**: `UT-EDI-TestLineEditorApply`

**Package**: `editor`

**File**: `internal/editor/line_editor_test.go:10`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Run
5. Call Join
   ... and 5 more steps

---

### TestLineEditorApplySingleLineEdit

**ID**: `UT-EDI-TestLineEditorApplySingleLineEdit`

**Package**: `editor`

**File**: `internal/editor/line_editor_test.go:452`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Join
5. Call WriteFile
   ... and 5 more steps

---

### TestLineEditorCheckOverlaps

**ID**: `UT-EDI-TestLineEditorCheckOverlaps`

**Package**: `editor`

**File**: `internal/editor/line_editor_test.go:206`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call checkOverlaps
3. Call Error
4. Call Errorf

---

### TestLineEditorDeleteLines

**ID**: `UT-EDI-TestLineEditorDeleteLines`

**Package**: `editor`

**File**: `internal/editor/line_editor_test.go:293`

**Description**: No description provided

**Test Steps**:
1. Call DeleteLines
2. Call Errorf
3. Call Errorf

---

### TestLineEditorEmptyContent

**ID**: `UT-EDI-TestLineEditorEmptyContent`

**Package**: `editor`

**File**: `internal/editor/line_editor_test.go:533`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Join
5. Call WriteFile
   ... and 3 more steps

---

### TestLineEditorGetLineRange

**ID**: `UT-EDI-TestLineEditorGetLineRange`

**Package**: `editor`

**File**: `internal/editor/line_editor_test.go:334`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call GetLineRange
3. Call Errorf
4. Call Errorf

---

### TestLineEditorGetStats

**ID**: `UT-EDI-TestLineEditorGetStats`

**Package**: `editor`

**File**: `internal/editor/line_editor_test.go:394`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call GetStats
3. Call Errorf
4. Call Errorf
5. Call Errorf

---

### TestLineEditorInsertLines

**ID**: `UT-EDI-TestLineEditorInsertLines`

**Package**: `editor`

**File**: `internal/editor/line_editor_test.go:272`

**Description**: No description provided

**Test Steps**:
1. Call InsertLines
2. Call Errorf
3. Call Errorf

---

### TestLineEditorLargeFile

**ID**: `UT-EDI-TestLineEditorLargeFile`

**Package**: `editor`

**File**: `internal/editor/line_editor_test.go:488`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Join
5. Call Repeat
   ... and 5 more steps

---

### TestLineEditorReplaceLines

**ID**: `UT-EDI-TestLineEditorReplaceLines`

**Package**: `editor`

**File**: `internal/editor/line_editor_test.go:313`

**Description**: No description provided

**Test Steps**:
1. Call ReplaceLines
2. Call Errorf
3. Call Errorf

---

### TestLineEditorValidateLineEdit

**ID**: `UT-EDI-TestLineEditorValidateLineEdit`

**Package**: `editor`

**File**: `internal/editor/line_editor_test.go:149`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call validateLineEdit
3. Call Error
4. Call Errorf

---

### TestListExecutions

**ID**: `UT-SHE-TestListExecutions`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:530`

**Description**: TestListExecutions tests listing running executions

**Test Steps**:
1. Call Sprintf
2. Call ExecuteAsync
3. Call Background
4. Call NoError
5. Call Sleep
   ... and 2 more steps

---

### TestListProjects

**ID**: `UT-PRO-TestListProjects`

**Package**: `project`

**File**: `internal/project/manager_test.go:71`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call NoError
3. Call RemoveAll
4. Call MkdirTemp
5. Call NoError
   ... and 5 more steps

---

### TestListSnapshots

**ID**: `UT-SNA-TestListSnapshots`

**Package**: `snapshots`

**File**: `internal/workflow/snapshots/snapshots_test.go:276`

**Description**: TestListSnapshots tests listing snapshots

**Test Steps**:
1. Call Fatalf
2. Call Background
3. Call CreateSnapshot
4. Call Fatalf
5. Call Sleep
   ... and 5 more steps

---

### TestListSnapshots_WithFilter

**ID**: `UT-SNA-TestListSnapshots_WithFilter`

**Package**: `snapshots`

**File**: `internal/workflow/snapshots/snapshots_test.go:319`

**Description**: TestListSnapshots_WithFilter tests filtering snapshots

**Test Steps**:
1. Call Fatalf
2. Call Background
3. Call CreateSnapshot
4. Call Fatalf
5. Call CreateSnapshot
   ... and 5 more steps

---

### TestListSnapshots_WithLimit

**ID**: `UT-SNA-TestListSnapshots_WithLimit`

**Package**: `snapshots`

**File**: `internal/workflow/snapshots/snapshots_test.go:370`

**Description**: TestListSnapshots_WithLimit tests limiting results

**Test Steps**:
1. Call Fatalf
2. Call Background
3. Call CreateSnapshot
4. Call Fatalf
5. Call Sleep
   ... and 3 more steps

---

### TestLlamaCPPProviderIntegration

**ID**: `UT-LLM-TestLlamaCPPProviderIntegration`

**Package**: `llm`

**File**: `internal/llm/integration_test.go:15`

**Description**: TestLlamaCPPProviderIntegration tests the Llama.cpp provider with integration

**Test Steps**:
1. Call Skipf
2. Call Background
3. Call IsAvailable
4. Call Skip
5. Call GetModels
   ... and 5 more steps

---

### TestLoad

**ID**: `UT-CON-TestLoad`

**Package**: `config`

**File**: `internal/config/config_test.go:13`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call WriteFile
4. Call NoError
5. Call Getenv
   ... and 5 more steps

---

### TestLoadTemplate

**ID**: `UT-NOT-TestLoadTemplate`

**Package**: `notification`

**File**: `internal/notification/engine_test.go:49`

**Description**: No description provided

**Test Steps**:
1. Call LoadTemplate
2. Call NoError
3. Call LoadTemplate
4. Call Error

---

### TestMapCodebase

**ID**: `UT-MAP-TestMapCodebase`

**Package**: `mapping`

**File**: `internal/tools/mapping/mapping_test.go:520`

**Description**: TestMapCodebase tests codebase mapping

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call WriteFile
4. Call NoError
5. Call Background
   ... and 5 more steps

---

### TestMapFile

**ID**: `UT-MAP-TestMapFile`

**Package**: `mapping`

**File**: `internal/tools/mapping/mapping_test.go:474`

**Description**: TestMapFile tests file mapping

**Test Steps**:
1. Call TempDir
2. Call Run
3. Call Join
4. Call WriteFile
5. Call NoError
   ... and 5 more steps

---

### TestMapOptions

**ID**: `UT-MAP-TestMapOptions`

**Package**: `mapping`

**File**: `internal/tools/mapping/mapping_test.go:727`

**Description**: TestMapOptions tests MapOptions

**Test Steps**:
1. Call Run
2. Call True
3. Call Equal
4. Call Greater
5. Call Contains
   ... and 5 more steps

---

### TestMaxChannel

**ID**: `UT-NOT-TestMaxChannel`

**Package**: `notification`

**File**: `internal/notification/engine_test.go:128`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call Equal
3. Call GetName
4. Call True
5. Call IsEnabled
   ... and 4 more steps

---

### TestMergeReasoningConfigs_NilCases

**ID**: `UT-LLM-TestMergeReasoningConfigs_NilCases`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:550`

**Description**: Test MergeReasoningConfigs

**Test Steps**:
1. Call Equal
2. Call Equal

---

### TestMergeReasoningConfigs_OverrideValues

**ID**: `UT-LLM-TestMergeReasoningConfigs_OverrideValues`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:562`

**Description**: No description provided

**Test Steps**:
1. Call True
2. Call Equal
3. Call Equal
4. Call Equal

---

### TestMergeReasoningConfigs_PreservesBaseWhenOverrideEmpty

**ID**: `UT-LLM-TestMergeReasoningConfigs_PreservesBaseWhenOverrideEmpty`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:582`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call Equal

---

### TestMessageConversion

**ID**: `UT-COM-TestMessageConversion`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:532`

**Description**: Test 20: Message Conversion

**Test Steps**:
1. Call Equal
2. Call Equal
3. Call Equal
4. Call Equal

---

### TestMessageFormat

**ID**: `UT-GIT-TestMessageFormat`

**Package**: `git`

**File**: `internal/tools/git/git_test.go:695`

**Description**: TestMessageFormat tests message formatting

**Test Steps**:
1. Call Run
2. Call FormatMessage
3. Call Errorf

---

### TestMessageGenerator

**ID**: `UT-GIT-TestMessageGenerator`

**Package**: `git`

**File**: `internal/tools/git/git_test.go:566`

**Description**: TestMessageGenerator tests message generation

**Test Steps**:
1. Call Run
2. Call New
3. Call Generate
4. Call Background
5. Call Fatalf
   ... and 5 more steps

---

### TestMetadataPersistence

**ID**: `UT-SNA-TestMetadataPersistence`

**Package**: `snapshots`

**File**: `internal/workflow/snapshots/snapshots_test.go:743`

**Description**: TestMetadataPersistence tests metadata persistence across manager instances

**Test Steps**:
1. Call Fatalf
2. Call Background
3. Call CreateSnapshot
4. Call Fatalf
5. Call Fatalf
   ... and 5 more steps

---

### TestMetrics

**ID**: `UT-AUT-TestMetrics`

**Package**: `autonomy`

**File**: `internal/workflow/autonomy/autonomy_test.go:520`

**Description**: TestMetrics tests metrics tracking

**Test Steps**:
1. Call RecordPermissionCheck
2. Call RecordPermissionCheck
3. Call RecordModeChange
4. Call RecordExecution
5. Call GetStats
   ... and 5 more steps

---

### TestMiddlewareOrder

**ID**: `UT-SER-TestMiddlewareOrder`

**Package**: `server`

**File**: `internal/server/server_test.go:208`

**Description**: No description provided

**Test Steps**:
1. Call NotNil

---

### TestMockAgent

**ID**: `UT-AGE-TestMockAgent`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:884`

**Description**: No description provided

**Test Steps**:
1. Call ID
2. Call Errorf
3. Call ID
4. Call NewTask
5. Call Execute
   ... and 5 more steps

---

### TestMockDiscordServer

**ID**: `UT-TES-TestMockDiscordServer`

**Package**: `testutil`

**File**: `internal/notification/testutil/mock_servers_test.go:192`

**Description**: No description provided

**Test Steps**:
1. Call Close
2. Call Marshal
3. Call Post
4. Call NewReader
5. Call NoError
   ... and 4 more steps

---

### TestMockDiscordServer_InvalidJSON

**ID**: `UT-TES-TestMockDiscordServer_InvalidJSON`

**Package**: `testutil`

**File**: `internal/notification/testutil/mock_servers_test.go:241`

**Description**: No description provided

**Test Steps**:
1. Call Close
2. Call Post
3. Call NewReader
4. Call NoError
5. Call Equal
   ... and 2 more steps

---

### TestMockDiscordServer_InvalidMethod

**ID**: `UT-TES-TestMockDiscordServer_InvalidMethod`

**Package**: `testutil`

**File**: `internal/notification/testutil/mock_servers_test.go:228`

**Description**: No description provided

**Test Steps**:
1. Call Close
2. Call Get
3. Call NoError
4. Call Equal
5. Call Equal
   ... and 1 more steps

---

### TestMockDiscordServer_MultipleRequests

**ID**: `UT-TES-TestMockDiscordServer_MultipleRequests`

**Package**: `testutil`

**File**: `internal/notification/testutil/mock_servers_test.go:254`

**Description**: No description provided

**Test Steps**:
1. Call Close
2. Call Marshal
3. Call Post
4. Call NewReader
5. Call Equal
   ... and 3 more steps

---

### TestMockDiscordServer_Reset

**ID**: `UT-TES-TestMockDiscordServer_Reset`

**Package**: `testutil`

**File**: `internal/notification/testutil/mock_servers_test.go:212`

**Description**: No description provided

**Test Steps**:
1. Call Close
2. Call Marshal
3. Call Post
4. Call NewReader
5. Call Equal
   ... and 4 more steps

---

### TestMockServers_ThreadSafety

**ID**: `UT-TES-TestMockServers_ThreadSafety`

**Package**: `testutil`

**File**: `internal/notification/testutil/mock_servers_test.go:271`

**Description**: Test thread safety

**Test Steps**:
1. Call Run
2. Call Close
3. Call Marshal
4. Call Post
5. Call NewReader
   ... and 5 more steps

---

### TestMockSlackServer

**ID**: `UT-TES-TestMockSlackServer`

**Package**: `testutil`

**File**: `internal/notification/testutil/mock_servers_test.go:13`

**Description**: No description provided

**Test Steps**:
1. Call Close
2. Call Marshal
3. Call Post
4. Call NewReader
5. Call NoError
   ... and 5 more steps

---

### TestMockSlackServer_InvalidJSON

**ID**: `UT-TES-TestMockSlackServer_InvalidJSON`

**Package**: `testutil`

**File**: `internal/notification/testutil/mock_servers_test.go:68`

**Description**: No description provided

**Test Steps**:
1. Call Close
2. Call Post
3. Call NewReader
4. Call NoError
5. Call Equal
   ... and 2 more steps

---

### TestMockSlackServer_InvalidMethod

**ID**: `UT-TES-TestMockSlackServer_InvalidMethod`

**Package**: `testutil`

**File**: `internal/notification/testutil/mock_servers_test.go:55`

**Description**: No description provided

**Test Steps**:
1. Call Close
2. Call Get
3. Call NoError
4. Call Equal
5. Call Equal
   ... and 1 more steps

---

### TestMockSlackServer_MultipleRequests

**ID**: `UT-TES-TestMockSlackServer_MultipleRequests`

**Package**: `testutil`

**File**: `internal/notification/testutil/mock_servers_test.go:81`

**Description**: No description provided

**Test Steps**:
1. Call Close
2. Call Marshal
3. Call Post
4. Call NewReader
5. Call Equal
   ... and 3 more steps

---

### TestMockSlackServer_Reset

**ID**: `UT-TES-TestMockSlackServer_Reset`

**Package**: `testutil`

**File**: `internal/notification/testutil/mock_servers_test.go:39`

**Description**: No description provided

**Test Steps**:
1. Call Close
2. Call Marshal
3. Call Post
4. Call NewReader
5. Call Equal
   ... and 4 more steps

---

### TestMockTelegramServer

**ID**: `UT-TES-TestMockTelegramServer`

**Package**: `testutil`

**File**: `internal/notification/testutil/mock_servers_test.go:100`

**Description**: No description provided

**Test Steps**:
1. Call Close
2. Call Marshal
3. Call Post
4. Call NewReader
5. Call NoError
   ... and 5 more steps

---

### TestMockTelegramServer_InvalidJSON

**ID**: `UT-TES-TestMockTelegramServer_InvalidJSON`

**Package**: `testutil`

**File**: `internal/notification/testutil/mock_servers_test.go:160`

**Description**: No description provided

**Test Steps**:
1. Call Close
2. Call Post
3. Call NewReader
4. Call NoError
5. Call Equal
   ... and 2 more steps

---

### TestMockTelegramServer_InvalidMethod

**ID**: `UT-TES-TestMockTelegramServer_InvalidMethod`

**Package**: `testutil`

**File**: `internal/notification/testutil/mock_servers_test.go:147`

**Description**: No description provided

**Test Steps**:
1. Call Close
2. Call Get
3. Call NoError
4. Call Equal
5. Call Equal
   ... and 1 more steps

---

### TestMockTelegramServer_MessageIDIncrement

**ID**: `UT-TES-TestMockTelegramServer_MessageIDIncrement`

**Package**: `testutil`

**File**: `internal/notification/testutil/mock_servers_test.go:173`

**Description**: No description provided

**Test Steps**:
1. Call Close
2. Call Marshal
3. Call Post
4. Call NewReader
5. Call Decode
   ... and 2 more steps

---

### TestMockTelegramServer_Reset

**ID**: `UT-TES-TestMockTelegramServer_Reset`

**Package**: `testutil`

**File**: `internal/notification/testutil/mock_servers_test.go:131`

**Description**: No description provided

**Test Steps**:
1. Call Close
2. Call Marshal
3. Call Post
4. Call NewReader
5. Call Equal
   ... and 4 more steps

---

### TestMockTranscriber

**ID**: `UT-VOI-TestMockTranscriber`

**Package**: `voice`

**File**: `internal/tools/voice/voice_test.go:292`

**Description**: TestMockTranscriber tests the mock transcriber

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Join
5. Call WriteFile
   ... and 5 more steps

---

### TestMockTranscriber_CustomResponse

**ID**: `UT-VOI-TestMockTranscriber_CustomResponse`

**Package**: `voice`

**File**: `internal/tools/voice/voice_test.go:332`

**Description**: TestMockTranscriber_CustomResponse tests custom mock responses

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Join
5. Call WriteFile
   ... and 5 more steps

---

### TestModeCapabilities

**ID**: `UT-AUT-TestModeCapabilities`

**Package**: `autonomy`

**File**: `internal/workflow/autonomy/autonomy_test.go:10`

**Description**: TestModeCapabilities tests mode capability definitions

**Test Steps**:
1. Call Run
2. Call Errorf
3. Call Errorf
4. Call Errorf
5. Call Errorf
   ... and 2 more steps

---

### TestModeConstants

**ID**: `UT-SES-TestModeConstants`

**Package**: `session`

**File**: `internal/session/session_test.go:33`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call Equal
3. Call Equal
4. Call Equal

---

### TestModeController

**ID**: `UT-PLA-TestModeController`

**Package**: `planmode`

**File**: `internal/workflow/planmode/planmode_test.go:214`

**Description**: Test Mode Controller

**Test Steps**:
1. Call Run
2. Call Equal
3. Call GetMode
4. Call Run
5. Call TransitionTo
   ... and 5 more steps

---

### TestModeHistory

**ID**: `UT-AUT-TestModeHistory`

**Package**: `autonomy`

**File**: `internal/workflow/autonomy/autonomy_test.go:611`

**Description**: TestModeHistory tests mode change history tracking

**Test Steps**:
1. Call Fatalf
2. Call Background
3. Call SetMode
4. Call Fatalf
5. Call GetHistory
   ... and 2 more steps

---

### TestModeLevels

**ID**: `UT-AUT-TestModeLevels`

**Package**: `autonomy`

**File**: `internal/workflow/autonomy/autonomy_test.go:79`

**Description**: TestModeLevels tests mode level comparisons

**Test Steps**:
1. Call Run
2. Call Level
3. Call Errorf
4. Call Compare
5. Call Error
   ... and 4 more steps

---

### TestModeManager

**ID**: `UT-AUT-TestModeManager`

**Package**: `autonomy`

**File**: `internal/workflow/autonomy/autonomy_test.go:113`

**Description**: TestModeManager tests mode management

**Test Steps**:
1. Call Fatalf
2. Call Background
3. Call GetMode
4. Call Errorf
5. Call GetMode
   ... and 5 more steps

---

### TestModeValidation

**ID**: `UT-AUT-TestModeValidation`

**Package**: `autonomy`

**File**: `internal/workflow/autonomy/autonomy_test.go:54`

**Description**: TestModeValidation tests mode validation

**Test Steps**:
1. Call Run
2. Call IsValid
3. Call Errorf

---

### TestModelCapabilities

**ID**: `UT-VIS-TestModelCapabilities`

**Package**: `vision`

**File**: `internal/llm/vision/vision_test.go:284`

**Description**: TestModelCapabilities tests model capability checking

**Test Steps**:
1. Call Background
2. Call Run
3. Call SupportsVision
4. Call Error
5. Call Errorf
   ... and 1 more steps

---

### TestModelCapabilitiesConsistency

**ID**: `UT-EDI-TestModelCapabilitiesConsistency`

**Package**: `editor`

**File**: `internal/editor/model_formats_test.go:334`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Errorf

---

### TestModelFormatPreferencesCoverage

**ID**: `UT-EDI-TestModelFormatPreferencesCoverage`

**Package**: `editor`

**File**: `internal/editor/model_formats_test.go:304`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Errorf
3. Call Errorf

---

### TestModelManagerIntegration

**ID**: `UT-LLM-TestModelManagerIntegration`

**Package**: `llm`

**File**: `internal/llm/integration_test.go:117`

**Description**: TestModelManagerIntegration tests the model manager with integration

**Test Steps**:
1. Call RegisterProvider
2. Call Logf
3. Call SelectOptimalModel
4. Call Logf
5. Call Logf
   ... and 5 more steps

---

### TestModelSizeCalculation

**ID**: `UT-HAR-TestModelSizeCalculation`

**Package**: `hardware`

**File**: `internal/hardware/detector_test.go:113`

**Description**: TestModelSizeCalculation tests the model size calculation logic

**Test Steps**:
1. Call GetOptimalModelSize
2. Call GetOptimalModelSize
3. Call Errorf
4. Call Errorf
5. Call Logf

---

### TestMultiFileEditor_AtomicRollback

**ID**: `UT-MUL-TestMultiFileEditor_AtomicRollback`

**Package**: `multiedit`

**File**: `internal/tools/multiedit/multiedit_test.go:564`

**Description**: Test 18: Atomic Write Rollback

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call NoError
4. Call Join
5. Call Join
   ... and 5 more steps

---

### TestMultiFileEditor_ChecksumMismatch

**ID**: `UT-MUL-TestMultiFileEditor_ChecksumMismatch`

**Package**: `multiedit`

**File**: `internal/tools/multiedit/multiedit_test.go:369`

**Description**: Test 13: Checksum Verification

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call NoError
4. Call Join
5. Call WriteFile
   ... and 5 more steps

---

### TestMultiFileEditor_CreateFile

**ID**: `UT-MUL-TestMultiFileEditor_CreateFile`

**Package**: `multiedit`

**File**: `internal/tools/multiedit/multiedit_test.go:407`

**Description**: Test 14: Create Operation

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call NoError
4. Call BeginEdit
5. Call Background
   ... and 5 more steps

---

### TestMultiFileEditor_DeleteFile

**ID**: `UT-MUL-TestMultiFileEditor_DeleteFile`

**Package**: `multiedit`

**File**: `internal/tools/multiedit/multiedit_test.go:446`

**Description**: Test 15: Delete Operation

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call NoError
4. Call Join
5. Call WriteFile
   ... and 5 more steps

---

### TestMultiFileEditor_LargeFile

**ID**: `UT-MUL-TestMultiFileEditor_LargeFile`

**Package**: `multiedit`

**File**: `internal/tools/multiedit/multiedit_test.go:642`

**Description**: Test 20: Large File Handling

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call NoError
4. Call BeginEdit
5. Call Background
   ... and 5 more steps

---

### TestMultiFileEditor_MultiFileEdit

**ID**: `UT-MUL-TestMultiFileEditor_MultiFileEdit`

**Package**: `multiedit`

**File**: `internal/tools/multiedit/multiedit_test.go:186`

**Description**: Test 8: Multi-File Edit

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call NoError
4. Call Join
5. Call Join
   ... and 5 more steps

---

### TestMultiFileEditor_RollbackOnError

**ID**: `UT-MUL-TestMultiFileEditor_RollbackOnError`

**Package**: `multiedit`

**File**: `internal/tools/multiedit/multiedit_test.go:247`

**Description**: Test 9: Rollback on Error

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call NoError
4. Call Join
5. Call WriteFile
   ... and 5 more steps

---

### TestMultilineCommand

**ID**: `UT-SHE-TestMultilineCommand`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:585`

**Description**: TestMultilineCommand tests execution of multiline commands

**Test Steps**:
1. Call Execute
2. Call Background
3. Call NoError
4. Call Equal
5. Call Contains
   ... and 2 more steps

---

### TestNewAnthropicProvider

**ID**: `UT-LLM-TestNewAnthropicProvider`

**Package**: `llm`

**File**: `internal/llm/anthropic_provider_test.go:17`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Setenv
3. Call Unsetenv
4. Call Unsetenv
5. Call Error
   ... and 5 more steps

---

### TestNewAuthService

**ID**: `UT-AUT-TestNewAuthService`

**Package**: `auth`

**File**: `internal/auth/auth_test.go:84`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call Equal
3. Call Equal

---

### TestNewAutoCommitCoordinator

**ID**: `UT-GIT-TestNewAutoCommitCoordinator`

**Package**: `git`

**File**: `internal/tools/git/git_test.go:106`

**Description**: TestNewAutoCommitCoordinator tests creating a new coordinator

**Test Steps**:
1. Call Run
2. Call New
3. Call Fatalf
4. Call Fatal
5. Call Errorf
   ... and 5 more steps

---

### TestNewBaseAgent

**ID**: `UT-AGE-TestNewBaseAgent`

**Package**: `agent`

**File**: `internal/agent/agent_test.go:12`

**Description**: No description provided

**Test Steps**:
1. Call ID
2. Call Errorf
3. Call ID
4. Call Type
5. Call Errorf
   ... and 5 more steps

---

### TestNewBedrockProvider

**ID**: `UT-LLM-TestNewBedrockProvider`

**Package**: `llm`

**File**: `internal/llm/bedrock_provider_test.go:38`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Setenv
3. Call Unsetenv
4. Call Short
5. Call Skip
   ... and 5 more steps

---

### TestNewCircuitBreaker

**ID**: `UT-AGE-TestNewCircuitBreaker`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:14`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call Equal
3. Call Equal
4. Call Equal
5. Call Equal
   ... and 3 more steps

---

### TestNewCircuitBreakerManager

**ID**: `UT-AGE-TestNewCircuitBreakerManager`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:442`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call NotNil
3. Call Equal
4. Call Equal
5. Call Equal

---

### TestNewClient_Disabled

**ID**: `UT-RED-TestNewClient_Disabled`

**Package**: `redis`

**File**: `internal/redis/redis_test.go:12`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call NotNil
3. Call False
4. Call IsEnabled
5. Call Nil
   ... and 1 more steps

---

### TestNewClient_InvalidConfig

**ID**: `UT-RED-TestNewClient_InvalidConfig`

**Package**: `redis`

**File**: `internal/redis/redis_test.go:26`

**Description**: No description provided

**Test Steps**:
1. Call Error
2. Call Nil
3. Call Contains
4. Call Error

---

### TestNewCodeEditor

**ID**: `UT-EDI-TestNewCodeEditor`

**Package**: `editor`

**File**: `internal/editor/editor_test.go:9`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Error
3. Call Errorf
4. Call Error
5. Call GetFormat
   ... and 2 more steps

---

### TestNewCodingAgent

**ID**: `UT-TYP-TestNewCodingAgent`

**Package**: `types`

**File**: `internal/agent/types/coding_agent_test.go:16`

**Description**: TestNewCodingAgent tests coding agent creation

**Test Steps**:
1. Call Run
2. Call NewToolRegistry
3. Call NoError
4. Call NoError
5. Call NotNil
   ... and 5 more steps

---

### TestNewDebuggingAgent

**ID**: `UT-TYP-TestNewDebuggingAgent`

**Package**: `types`

**File**: `internal/agent/types/debugging_agent_test.go:17`

**Description**: TestNewDebuggingAgent tests debugging agent creation

**Test Steps**:
1. Call Run
2. Call NewToolRegistry
3. Call NoError
4. Call NoError
5. Call NotNil
   ... and 5 more steps

---

### TestNewDiscordChannel

**ID**: `UT-NOT-TestNewDiscordChannel`

**Package**: `notification`

**File**: `internal/notification/discord_test.go:13`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call NotNil
3. Call Equal
4. Call GetName
5. Call Equal
   ... and 3 more steps

---

### TestNewEmailChannel

**ID**: `UT-NOT-TestNewEmailChannel`

**Package**: `notification`

**File**: `internal/notification/email_test.go:11`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Equal
3. Call IsEnabled
4. Call Equal
5. Call GetName

---

### TestNewEventBus

**ID**: `UT-EVE-TestNewEventBus`

**Package**: `event`

**File**: `internal/event/bus_test.go:15`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call NotNil
3. Call Equal
4. Call IsAsync
5. Call Equal
   ... and 1 more steps

---

### TestNewEventNotificationHandler

**ID**: `UT-NOT-TestNewEventNotificationHandler`

**Package**: `notification`

**File**: `internal/notification/event_handler_test.go:13`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call NotNil

---

### TestNewExecutor

**ID**: `UT-WOR-TestNewExecutor`

**Package**: `workflow`

**File**: `internal/workflow/executor_test.go:12`

**Description**: No description provided

**Test Steps**:
1. Call NewManager
2. Call NotNil
3. Call Equal

---

### TestNewFileRanker

**ID**: `UT-REP-TestNewFileRanker`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:556`

**Description**: Test FileRanker

**Test Steps**:
1. Call Fatal
2. Call Error

---

### TestNewFileSystemTools

**ID**: `UT-FIL-TestNewFileSystemTools`

**Package**: `filesystem`

**File**: `internal/tools/filesystem/filesystem_test.go:15`

**Description**: TestNewFileSystemTools tests creating a new file system tools instance

**Test Steps**:
1. Call Run
2. Call Errorf
3. Call Error

---

### TestNewGeminiProvider

**ID**: `UT-LLM-TestNewGeminiProvider`

**Package**: `llm`

**File**: `internal/llm/gemini_provider_test.go:17`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Unsetenv
3. Call Unsetenv
4. Call Setenv
5. Call Unsetenv
   ... and 5 more steps

---

### TestNewGroqProvider

**ID**: `UT-LLM-TestNewGroqProvider`

**Package**: `llm`

**File**: `internal/llm/groq_provider_test.go:18`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Setenv
3. Call Unsetenv
4. Call Unsetenv
5. Call Error
   ... and 5 more steps

---

### TestNewLogoProcessor

**ID**: `UT-LOG-TestNewLogoProcessor`

**Package**: `logo`

**File**: `internal/logo/processor_test.go:12`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call Equal
3. Call Equal
4. Call Equal
5. Call Equal
   ... and 1 more steps

---

### TestNewMCPServer

**ID**: `UT-MCP-TestNewMCPServer`

**Package**: `mcp`

**File**: `internal/mcp/server_test.go:10`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call NotNil
3. Call NotNil
4. Call Equal
5. Call GetSessionCount
   ... and 2 more steps

---

### TestNewManager

**ID**: `UT-PRO-TestNewManager`

**Package**: `project`

**File**: `internal/project/manager_test.go:12`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call NotNil
3. Call Nil

---

### TestNewManager

**ID**: `UT-SNA-TestNewManager`

**Package**: `snapshots`

**File**: `internal/workflow/snapshots/snapshots_test.go:83`

**Description**: TestNewManager tests manager creation

**Test Steps**:
1. Call Fatalf
2. Call Fatal
3. Call Errorf

---

### TestNewManager_NotGitRepo

**ID**: `UT-SNA-TestNewManager_NotGitRepo`

**Package**: `snapshots`

**File**: `internal/workflow/snapshots/snapshots_test.go:102`

**Description**: TestNewManager_NotGitRepo tests manager creation on non-git directory

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Error
5. Call Contains
   ... and 2 more steps

---

### TestNewNotificationEngine

**ID**: `UT-NOT-TestNewNotificationEngine`

**Package**: `notification`

**File**: `internal/notification/engine_test.go:10`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call NotNil
3. Call NotNil
4. Call NotNil

---

### TestNewNotificationQueue

**ID**: `UT-NOT-TestNewNotificationQueue`

**Package**: `notification`

**File**: `internal/notification/queue_test.go:12`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call Equal
3. Call Equal
4. Call True
5. Call IsEmpty

---

### TestNewPlanningAgent

**ID**: `UT-TYP-TestNewPlanningAgent`

**Package**: `types`

**File**: `internal/agent/types/planning_agent_test.go:64`

**Description**: TestNewPlanningAgent tests planning agent creation

**Test Steps**:
1. Call Run
2. Call NoError
3. Call NotNil
4. Call Equal
5. Call ID
   ... and 5 more steps

---

### TestNewQwenProvider

**ID**: `UT-LLM-TestNewQwenProvider`

**Package**: `llm`

**File**: `internal/llm/qwen_provider_test.go:15`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Error
3. Call Contains
4. Call Error
5. Call Nil
   ... and 5 more steps

---

### TestNewRateLimitedChannel

**ID**: `UT-NOT-TestNewRateLimitedChannel`

**Package**: `notification`

**File**: `internal/notification/ratelimit_test.go:137`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call NotNil
3. Call NotNil
4. Call NotNil

---

### TestNewRateLimiter

**ID**: `UT-NOT-TestNewRateLimiter`

**Package**: `notification`

**File**: `internal/notification/ratelimit_test.go:12`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call Equal
3. Call Equal
4. Call Equal

---

### TestNewReasoningConfig_Claude_Opus

**ID**: `UT-LLM-TestNewReasoningConfig_Claude_Opus`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:37`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call True
3. Call True
4. Call Equal
5. Call Equal
   ... and 1 more steps

---

### TestNewReasoningConfig_DeepSeek_R1

**ID**: `UT-LLM-TestNewReasoningConfig_DeepSeek_R1`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:48`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call True
3. Call Equal
4. Call Equal
5. Call Equal

---

### TestNewReasoningConfig_OpenAI_O1

**ID**: `UT-LLM-TestNewReasoningConfig_OpenAI_O1`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:26`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call True
3. Call True
4. Call Equal
5. Call Equal
   ... and 1 more steps

---

### TestNewReasoningConfig_QwQ_32B

**ID**: `UT-LLM-TestNewReasoningConfig_QwQ_32B`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:58`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call True
3. Call Equal
4. Call Equal
5. Call Equal

---

### TestNewRepoCache

**ID**: `UT-REP-TestNewRepoCache`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:676`

**Description**: Test Cache

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call Fatalf
4. Call Fatal
5. Call Stat
   ... and 2 more steps

---

### TestNewRepoMap

**ID**: `UT-REP-TestNewRepoMap`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:12`

**Description**: Test RepoMap creation and initialization

**Test Steps**:
1. Call TempDir
2. Call Fatalf
3. Call Fatal
4. Call Errorf

---

### TestNewRepoMapInvalidPath

**ID**: `UT-REP-TestNewRepoMapInvalidPath`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:31`

**Description**: No description provided

**Test Steps**:
1. Call Fatal

---

### TestNewResult

**ID**: `UT-TAS-TestNewResult`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:340`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call Equal
3. Call False
4. Call NotNil
5. Call NotNil
   ... and 3 more steps

---

### TestNewReviewAgent

**ID**: `UT-TYP-TestNewReviewAgent`

**Package**: `types`

**File**: `internal/agent/types/review_agent_test.go:17`

**Description**: TestNewReviewAgent tests review agent creation

**Test Steps**:
1. Call Run
2. Call NewToolRegistry
3. Call NoError
4. Call NoError
5. Call NotNil
   ... and 5 more steps

---

### TestNewServer

**ID**: `UT-SER-TestNewServer`

**Package**: `server`

**File**: `internal/server/server_test.go:38`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call Equal
3. Call Equal
4. Call Equal
5. Call NotNil

---

### TestNewServer_DebugMode

**ID**: `UT-SER-TestNewServer_DebugMode`

**Package**: `server`

**File**: `internal/server/server_test.go:50`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call Equal
3. Call Mode

---

### TestNewServer_ReleaseMode

**ID**: `UT-SER-TestNewServer_ReleaseMode`

**Package**: `server`

**File**: `internal/server/server_test.go:61`

**Description**: No description provided

**Test Steps**:
1. Call SetMode
2. Call NotNil
3. Call Equal
4. Call Mode

---

### TestNewSlackChannel

**ID**: `UT-NOT-TestNewSlackChannel`

**Package**: `notification`

**File**: `internal/notification/slack_test.go:15`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Equal
3. Call IsEnabled
4. Call Equal
5. Call GetName

---

### TestNewTagExtractor

**ID**: `UT-REP-TestNewTagExtractor`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:520`

**Description**: Test TagExtractor

**Test Steps**:
1. Call Fatal
2. Call Error

---

### TestNewTask

**ID**: `UT-TAS-TestNewTask`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:15`

**Description**: No description provided

**Test Steps**:
1. Call NotEmpty
2. Call Equal
3. Call Equal
4. Call Equal
5. Call Equal
   ... and 5 more steps

---

### TestNewTaskAllPriorities

**ID**: `UT-TAS-TestNewTaskAllPriorities`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:56`

**Description**: No description provided

**Test Steps**:
1. Call Equal

---

### TestNewTaskAllTypes

**ID**: `UT-TAS-TestNewTaskAllTypes`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:36`

**Description**: No description provided

**Test Steps**:
1. Call Equal

---

### TestNewTelegramChannel

**ID**: `UT-NOT-TestNewTelegramChannel`

**Package**: `notification`

**File**: `internal/notification/telegram_test.go:16`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Equal
3. Call IsEnabled
4. Call Equal
5. Call GetName

---

### TestNewTestingAgent

**ID**: `UT-TYP-TestNewTestingAgent`

**Package**: `types`

**File**: `internal/agent/types/testing_agent_test.go:17`

**Description**: TestNewTestingAgent tests testing agent creation

**Test Steps**:
1. Call Run
2. Call NewToolRegistry
3. Call NoError
4. Call NoError
5. Call NotNil
   ... and 5 more steps

---

### TestNewTokenTracker

**ID**: `UT-LLM-TestNewTokenTracker`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:156`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call Equal
3. Call NotNil
4. Call NotNil
5. Call NotNil
   ... and 3 more steps

---

### TestNewTreeSitterParser

**ID**: `UT-REP-TestNewTreeSitterParser`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:432`

**Description**: Test TreeSitterParser

**Test Steps**:
1. Call Fatal
2. Call SupportedLanguages
3. Call Errorf

---

### TestNewVertexAIProvider

**ID**: `UT-LLM-TestNewVertexAIProvider`

**Package**: `llm`

**File**: `internal/llm/vertexai_provider_test.go:34`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Unsetenv
3. Call Unsetenv
4. Call Unsetenv
5. Call Unsetenv
   ... and 5 more steps

---

### TestNewWebTools_CustomConfig

**ID**: `UT-WEB-TestNewWebTools_CustomConfig`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:30`

**Description**: Test 2: WebTools creation with custom config

**Test Steps**:
1. Call NoError
2. Call NotNil
3. Call Equal
4. Call Equal
5. Call Close

---

### TestNewWebTools_DefaultConfig

**ID**: `UT-WEB-TestNewWebTools_DefaultConfig`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:16`

**Description**: Test 1: WebTools creation with default config

**Test Steps**:
1. Call NoError
2. Call NotNil
3. Call NotNil
4. Call NotNil
5. Call NotNil
   ... and 3 more steps

---

### TestNewWorkflow

**ID**: `UT-AGE-TestNewWorkflow`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:14`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call NotEmpty
3. Call Equal
4. Call Equal
5. Call Equal
   ... and 5 more steps

---

### TestNew_InvalidConfig

**ID**: `UT-DAT-TestNew_InvalidConfig`

**Package**: `database`

**File**: `internal/database/database_test.go:27`

**Description**: No description provided

**Test Steps**:
1. Call Error
2. Call Nil
3. Call Contains
4. Call Error

---

### TestNodeOperations

**ID**: `UT-MAP-TestNodeOperations`

**Package**: `mapping`

**File**: `internal/tools/mapping/mapping_test.go:677`

**Description**: TestNodeOperations tests Node operations

**Test Steps**:
1. Call Run
2. Call FindChild
3. Call NotNil
4. Call Equal
5. Call Run
   ... and 5 more steps

---

### TestNotificationQueue_Clear

**ID**: `UT-NOT-TestNotificationQueue_Clear`

**Package**: `notification`

**File**: `internal/notification/queue_test.go:95`

**Description**: No description provided

**Test Steps**:
1. Call Enqueue
2. Call Enqueue
3. Call Equal
4. Call Size
5. Call Clear
   ... and 2 more steps

---

### TestNotificationQueue_Dequeue

**ID**: `UT-NOT-TestNotificationQueue_Dequeue`

**Package**: `notification`

**File**: `internal/notification/queue_test.go:62`

**Description**: No description provided

**Test Steps**:
1. Call Enqueue
2. Call Dequeue
3. Call NotNil
4. Call Equal
5. Call Equal
   ... and 5 more steps

---

### TestNotificationQueue_DequeueEmpty

**ID**: `UT-NOT-TestNotificationQueue_DequeueEmpty`

**Package**: `notification`

**File**: `internal/notification/queue_test.go:87`

**Description**: No description provided

**Test Steps**:
1. Call Dequeue
2. Call Nil

---

### TestNotificationQueue_Enqueue

**ID**: `UT-NOT-TestNotificationQueue_Enqueue`

**Package**: `notification`

**File**: `internal/notification/queue_test.go:22`

**Description**: No description provided

**Test Steps**:
1. Call Enqueue
2. Call NoError
3. Call Equal
4. Call Size
5. Call False
   ... and 3 more steps

---

### TestNotificationQueue_EnqueueFull

**ID**: `UT-NOT-TestNotificationQueue_EnqueueFull`

**Package**: `notification`

**File**: `internal/notification/queue_test.go:42`

**Description**: No description provided

**Test Steps**:
1. Call Enqueue
2. Call Enqueue
3. Call Enqueue
4. Call Error
5. Call Contains
   ... and 1 more steps

---

### TestNotificationQueue_GetQueueItems

**ID**: `UT-NOT-TestNotificationQueue_GetQueueItems`

**Package**: `notification`

**File**: `internal/notification/queue_test.go:148`

**Description**: No description provided

**Test Steps**:
1. Call Enqueue
2. Call Enqueue
3. Call GetQueueItems
4. Call Equal
5. Call Equal
   ... and 1 more steps

---

### TestNotificationQueue_ResetStats

**ID**: `UT-NOT-TestNotificationQueue_ResetStats`

**Package**: `notification`

**File**: `internal/notification/queue_test.go:164`

**Description**: No description provided

**Test Steps**:
1. Call Enqueue
2. Call GetStats
3. Call Equal
4. Call ResetStats
5. Call GetStats
   ... and 1 more steps

---

### TestNotificationQueue_Worker

**ID**: `UT-NOT-TestNotificationQueue_Worker`

**Package**: `notification`

**File**: `internal/notification/queue_test.go:113`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call Start
3. Call Stop
4. Call Enqueue
5. Call Sleep
   ... and 5 more steps

---

### TestOllamaProviderIntegration

**ID**: `UT-LLM-TestOllamaProviderIntegration`

**Package**: `llm`

**File**: `internal/llm/integration_test.go:67`

**Description**: TestOllamaProviderIntegration tests the Ollama provider with integration

**Test Steps**:
1. Call Skipf
2. Call Background
3. Call IsAvailable
4. Call Skip
5. Call GetModels
   ... and 5 more steps

---

### TestOptimizeReasoningConfig_DisabledConfig

**ID**: `UT-LLM-TestOptimizeReasoningConfig_DisabledConfig`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:504`

**Description**: Test OptimizeReasoningConfig

**Test Steps**:
1. Call Background
2. Call Equal

---

### TestOptimizeReasoningConfig_PreservesExistingBudget

**ID**: `UT-LLM-TestOptimizeReasoningConfig_PreservesExistingBudget`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:538`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call Equal

---

### TestOptimizeReasoningConfig_SetsBudgetBasedOnEffort

**ID**: `UT-LLM-TestOptimizeReasoningConfig_SetsBudgetBasedOnEffort`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:513`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Background
3. Call Equal

---

### TestOptionPresenter

**ID**: `UT-PLA-TestOptionPresenter`

**Package**: `planmode`

**File**: `internal/workflow/planmode/planmode_test.go:501`

**Description**: Test Option Presenter

**Test Steps**:
1. Call Run
2. Call NewReader
3. Call Present
4. Call Background
5. Call NoError
   ... and 5 more steps

---

### TestOutputTruncation

**ID**: `UT-SHE-TestOutputTruncation`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:242`

**Description**: TestOutputTruncation tests output size limits

**Test Steps**:
1. Call Execute
2. Call Background
3. Call NoError
4. Call True
5. Call Contains

---

### TestParseDiff

**ID**: `UT-GIT-TestParseDiff`

**Package**: `git`

**File**: `internal/tools/git/git_test.go:658`

**Description**: TestParseDiff tests diff parsing

**Test Steps**:
1. Call Run
2. Call Fatalf
3. Call Errorf
4. Call Fatalf
5. Call Run
   ... and 1 more steps

---

### TestParseFile

**ID**: `UT-REP-TestParseFile`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:445`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call ParseFile
4. Call Fatalf
5. Call Fatal

---

### TestParseFileUnsupportedLanguage

**ID**: `UT-REP-TestParseFileUnsupportedLanguage`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:463`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Join
3. Call ParseFile
4. Call Error

---

### TestParsedTreeQuery

**ID**: `UT-MAP-TestParsedTreeQuery`

**Package**: `mapping`

**File**: `internal/tools/mapping/mapping_test.go:709`

**Description**: TestParsedTreeQuery tests tree querying

**Test Steps**:
1. Call Query
2. Call Len

---

### TestParser_ExtractMetadata

**ID**: `UT-WEB-TestParser_ExtractMetadata`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:169`

**Description**: Test 8: Metadata extraction

**Test Steps**:
1. Call Parse
2. Call NoError
3. Call Equal
4. Call Equal
5. Call Equal
   ... and 3 more steps

---

### TestParser_ExtractText

**ID**: `UT-WEB-TestParser_ExtractText`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:551`

**Description**: Test 24: Extract plain text

**Test Steps**:
1. Call ExtractText
2. Call NoError
3. Call Contains
4. Call Contains
5. Call Contains
   ... and 1 more steps

---

### TestParser_LinksAndImages

**ID**: `UT-WEB-TestParser_LinksAndImages`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:460`

**Description**: Test 19: Parser links and images

**Test Steps**:
1. Call Parse
2. Call NoError
3. Call Contains
4. Call Contains

---

### TestParser_Parse_HTMLToMarkdown

**ID**: `UT-WEB-TestParser_Parse_HTMLToMarkdown`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:138`

**Description**: Test 7: HTML to markdown conversion

**Test Steps**:
1. Call Parse
2. Call NoError
3. Call Contains
4. Call Contains
5. Call Contains
   ... and 2 more steps

---

### TestParser_RemoveScriptsAndStyles

**ID**: `UT-WEB-TestParser_RemoveScriptsAndStyles`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:394`

**Description**: Test 17: Remove scripts and styles

**Test Steps**:
1. Call Parse
2. Call NoError
3. Call Contains
4. Call NotContains
5. Call NotContains

---

### TestPathSanitization

**ID**: `UT-SHE-TestPathSanitization`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:370`

**Description**: TestPathSanitization tests path sanitization

**Test Steps**:
1. Call Run
2. Call NotContains

---

### TestPathValidator

**ID**: `UT-FIL-TestPathValidator`

**Package**: `filesystem`

**File**: `internal/tools/filesystem/filesystem_test.go:608`

**Description**: TestPathValidator tests path validation

**Test Steps**:
1. Call MkdirTemp
2. Call Fatal
3. Call RemoveAll
4. Call Join
5. Call Join
   ... and 5 more steps

---

### TestPermissionChecking

**ID**: `UT-AUT-TestPermissionChecking`

**Package**: `autonomy`

**File**: `internal/workflow/autonomy/autonomy_test.go:159`

**Description**: TestPermissionChecking tests permission logic for different modes

**Test Steps**:
1. Call Background
2. Call Run
3. Call Check
4. Call Fatalf
5. Call Errorf
   ... and 1 more steps

---

### TestPlanMode

**ID**: `UT-PLA-TestPlanMode`

**Package**: `planmode`

**File**: `internal/workflow/planmode/planmode_test.go:655`

**Description**: Test PlanMode

**Test Steps**:
1. Call Run
2. Call Equal
3. Call Equal
4. Call False
5. Call True
   ... and 5 more steps

---

### TestPlanModeWorkflow

**ID**: `UT-PLA-TestPlanModeWorkflow`

**Package**: `planmode`

**File**: `internal/workflow/planmode/planmode_test.go:602`

**Description**: Test Workflow

**Test Steps**:
1. Call Run
2. Call ExecuteWorkflow
3. Call Background
4. Call NoError
5. Call NotNil
   ... and 5 more steps

---

### TestPlanner

**ID**: `UT-PLA-TestPlanner`

**Package**: `planmode`

**File**: `internal/workflow/planmode/planmode_test.go:300`

**Description**: Test Planner

**Test Steps**:
1. Call Run
2. Call GeneratePlan
3. Call Background
4. Call NoError
5. Call NotNil
   ... and 5 more steps

---

### TestPlanningAgentCollaborate

**ID**: `UT-TYP-TestPlanningAgentCollaborate`

**Package**: `types`

**File**: `internal/agent/types/planning_agent_test.go:516`

**Description**: TestPlanningAgentCollaborate tests collaboration with other agents

**Test Steps**:
1. Call NoError
2. Call NewBaseAgent
3. Call Background
4. Call NewTask
5. Call Collaborate
   ... and 5 more steps

---

### TestPlanningAgentExecuteGeneratePlanError

**ID**: `UT-TYP-TestPlanningAgentExecuteGeneratePlanError`

**Package**: `types`

**File**: `internal/agent/types/planning_agent_test.go:433`

**Description**: TestPlanningAgentExecuteGeneratePlanError tests error during plan generation

**Test Steps**:
1. Call NoError
2. Call Background
3. Call NewTask
4. Call Execute
5. Call Error
   ... and 5 more steps

---

### TestPlanningAgentExecuteMissingRequirements

**ID**: `UT-TYP-TestPlanningAgentExecuteMissingRequirements`

**Package**: `types`

**File**: `internal/agent/types/planning_agent_test.go:401`

**Description**: TestPlanningAgentExecuteMissingRequirements tests error when requirements are missing

**Test Steps**:
1. Call NoError
2. Call Background
3. Call NewTask
4. Call Execute
5. Call Error
   ... and 5 more steps

---

### TestPlanningAgentExecuteParseSubtasksError

**ID**: `UT-TYP-TestPlanningAgentExecuteParseSubtasksError`

**Package**: `types`

**File**: `internal/agent/types/planning_agent_test.go:468`

**Description**: TestPlanningAgentExecuteParseSubtasksError tests error during subtask parsing

**Test Steps**:
1. Call NoError
2. Call Background
3. Call NewTask
4. Call Execute
5. Call Error
   ... and 5 more steps

---

### TestPlanningAgentExecuteSuccess

**ID**: `UT-TYP-TestPlanningAgentExecuteSuccess`

**Package**: `types`

**File**: `internal/agent/types/planning_agent_test.go:322`

**Description**: TestPlanningAgentExecuteSuccess tests successful plan generation and execution

**Test Steps**:
1. Call NoError
2. Call Background
3. Call NewTask
4. Call Execute
5. Call NoError
   ... and 5 more steps

---

### TestPlanningAgentInitialize

**ID**: `UT-TYP-TestPlanningAgentInitialize`

**Package**: `types`

**File**: `internal/agent/types/planning_agent_test.go:95`

**Description**: TestPlanningAgentInitialize tests agent initialization

**Test Steps**:
1. Call NoError
2. Call Background
3. Call Initialize
4. Call NoError
5. Call Equal
   ... and 1 more steps

---

### TestPlanningAgentShutdown

**ID**: `UT-TYP-TestPlanningAgentShutdown`

**Package**: `types`

**File**: `internal/agent/types/planning_agent_test.go:115`

**Description**: TestPlanningAgentShutdown tests agent shutdown

**Test Steps**:
1. Call NoError
2. Call Background
3. Call Shutdown
4. Call NoError
5. Call Equal
   ... and 1 more steps

---

### TestPlatformSpecificDetection

**ID**: `UT-HAR-TestPlatformSpecificDetection`

**Package**: `hardware`

**File**: `internal/hardware/detector_test.go:84`

**Description**: TestPlatformSpecificDetection tests platform-specific detection logic

**Test Steps**:
1. Call Detect
2. Call Fatalf
3. Call Log
4. Call Log
5. Call Log
   ... and 1 more steps

---

### TestPolicyBuilder

**ID**: `UT-COM-TestPolicyBuilder`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:510`

**Description**: Test 19: Policy Builder

**Test Steps**:
1. Call Build
2. Call AddRule
3. Call WithDefaultRules
4. Call WithMinAge
5. Call WithRecentCount
   ... and 5 more steps

---

### TestPolicyEngine_EvaluateAllowReads

**ID**: `UT-CON-TestPolicyEngine_EvaluateAllowReads`

**Package**: `confirmation`

**File**: `internal/tools/confirmation/confirmation_test.go:11`

**Description**: Test 1: PolicyEngine Evaluate - Allow reads

**Test Steps**:
1. Call SetPolicy
2. Call Fatalf
3. Call Evaluate
4. Call Fatalf
5. Call Errorf

---

### TestPolicyEngine_EvaluateDefaultAction

**ID**: `UT-CON-TestPolicyEngine_EvaluateDefaultAction`

**Package**: `confirmation`

**File**: `internal/tools/confirmation/confirmation_test.go:91`

**Description**: Test 3: PolicyEngine Evaluate - Default action for unmatched

**Test Steps**:
1. Call SetPolicy
2. Call Fatalf
3. Call Evaluate
4. Call Fatalf
5. Call Errorf

---

### TestPolicyEngine_EvaluateDenyDeletes

**ID**: `UT-CON-TestPolicyEngine_EvaluateDenyDeletes`

**Package**: `confirmation`

**File**: `internal/tools/confirmation/confirmation_test.go:51`

**Description**: Test 2: PolicyEngine Evaluate - Deny deletes

**Test Steps**:
1. Call SetPolicy
2. Call Fatalf
3. Call Evaluate
4. Call Fatalf
5. Call Errorf

---

### TestPolicyPresets

**ID**: `UT-COM-TestPolicyPresets`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:460`

**Description**: Test 16: Policy Presets

**Test Steps**:
1. Call Run
2. Call NotNil
3. Call NotEmpty
4. Call GetRules

---

### TestPolicyValidation_ConflictingPriorities

**ID**: `UT-CON-TestPolicyValidation_ConflictingPriorities`

**Package**: `confirmation`

**File**: `internal/tools/confirmation/confirmation_test.go:507`

**Description**: Test 15: Policy validation - Conflicting priorities

**Test Steps**:
1. Call Error
2. Call Contains
3. Call Error
4. Call Errorf

---

### TestPreviewEngine_Preview

**ID**: `UT-MUL-TestPreviewEngine_Preview`

**Package**: `multiedit`

**File**: `internal/tools/multiedit/multiedit_test.go:301`

**Description**: Test 10: Preview Generation

**Test Steps**:
1. Call Preview
2. Call Background
3. Call NoError
4. Call Len
5. Call Equal
   ... and 1 more steps

---

### TestPreviewFormatter_Format

**ID**: `UT-MUL-TestPreviewFormatter_Format`

**Package**: `multiedit`

**File**: `internal/tools/multiedit/multiedit_test.go:523`

**Description**: Test 17: Preview Formatter

**Test Steps**:
1. Call Format
2. Call NoError
3. Call Contains
4. Call Contains
5. Call Format
   ... and 5 more steps

---

### TestPromptFormatter_Format

**ID**: `UT-CON-TestPromptFormatter_Format`

**Package**: `confirmation`

**File**: `internal/tools/confirmation/confirmation_test.go:218`

**Description**: Test 8: PromptFormatter - Format prompt

**Test Steps**:
1. Call Format
2. Call Errorf
3. Call Contains
4. Call Error
5. Call Contains
   ... and 2 more steps

---

### TestProviderHealthIntegration

**ID**: `UT-LLM-TestProviderHealthIntegration`

**Package**: `llm`

**File**: `internal/llm/integration_test.go:254`

**Description**: TestProviderHealthIntegration tests provider health monitoring with integration

**Test Steps**:
1. Call RegisterProvider
2. Call Fatalf
3. Call Background
4. Call HealthCheck
5. Call Errorf
   ... and 2 more steps

---

### TestProviderManagerWithBudget

**ID**: `UT-LLM-TestProviderManagerWithBudget`

**Package**: `llm`

**File**: `internal/llm/provider_features_test.go:310`

**Description**: TestProviderManagerWithBudget tests ProviderManager with token budgets

**Test Steps**:
1. Call NotNil
2. Call NotNil
3. Call NotNil
4. Call GetTokenTracker
5. Call NotNil
   ... and 5 more steps

---

### TestQuickExecute

**ID**: `UT-SHE-TestQuickExecute`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:449`

**Description**: TestQuickExecute tests the convenience function

**Test Steps**:
1. Call NoError
2. Call Equal
3. Call Contains

---

### TestQuickExecuteWithTimeout

**ID**: `UT-SHE-TestQuickExecuteWithTimeout`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:457`

**Description**: TestQuickExecuteWithTimeout tests the timeout convenience function

**Test Steps**:
1. Call NoError
2. Call True

---

### TestQwenProviderIntegration

**ID**: `UT-LLM-TestQwenProviderIntegration`

**Package**: `llm`

**File**: `internal/llm/integration_test.go:178`

**Description**: TestQwenProviderIntegration tests the Qwen provider with integration

**Test Steps**:
1. Call Skipf
2. Call Background
3. Call IsAvailable
4. Call Skip
5. Call GetModels
   ... and 5 more steps

---

### TestQwenProvider_Close

**ID**: `UT-LLM-TestQwenProvider_Close`

**Package**: `llm`

**File**: `internal/llm/qwen_provider_test.go:382`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call Close
3. Call NoError

---

### TestQwenProvider_ErrorHandling

**ID**: `UT-LLM-TestQwenProvider_ErrorHandling`

**Package**: `llm`

**File**: `internal/llm/qwen_provider_test.go:394`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Set
4. Call Header
5. Call WriteHeader
   ... and 5 more steps

---

### TestQwenProvider_Generate

**ID**: `UT-LLM-TestQwenProvider_Generate`

**Package**: `llm`

**File**: `internal/llm/qwen_provider_test.go:235`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Set
4. Call Header
5. Call WriteHeader
   ... and 5 more steps

---

### TestQwenProvider_GenerateStream

**ID**: `UT-LLM-TestQwenProvider_GenerateStream`

**Package**: `llm`

**File**: `internal/llm/qwen_provider_test.go:300`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Set
4. Call Header
5. Call WriteHeader
   ... and 5 more steps

---

### TestQwenProvider_GetCapabilities

**ID**: `UT-LLM-TestQwenProvider_GetCapabilities`

**Package**: `llm`

**File**: `internal/llm/qwen_provider_test.go:124`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call GetCapabilities
3. Call Equal

---

### TestQwenProvider_GetHealth

**ID**: `UT-LLM-TestQwenProvider_GetHealth`

**Package**: `llm`

**File**: `internal/llm/qwen_provider_test.go:173`

**Description**: No description provided

**Test Steps**:
1. Call Set
2. Call Header
3. Call WriteHeader
4. Call Write
5. Call WriteHeader
   ... and 5 more steps

---

### TestQwenProvider_GetModels

**ID**: `UT-LLM-TestQwenProvider_GetModels`

**Package**: `llm`

**File**: `internal/llm/qwen_provider_test.go:89`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call GetModels
3. Call NotEmpty
4. Call Equal
5. Call NotEmpty
   ... and 3 more steps

---

### TestQwenProvider_GetName

**ID**: `UT-LLM-TestQwenProvider_GetName`

**Package**: `llm`

**File**: `internal/llm/qwen_provider_test.go:78`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call Equal
3. Call GetName

---

### TestQwenProvider_GetType

**ID**: `UT-LLM-TestQwenProvider_GetType`

**Package**: `llm`

**File**: `internal/llm/qwen_provider_test.go:67`

**Description**: No description provided

**Test Steps**:
1. Call NoError
2. Call Equal
3. Call GetType

---

### TestQwenProvider_IsAvailable

**ID**: `UT-LLM-TestQwenProvider_IsAvailable`

**Package**: `llm`

**File**: `internal/llm/qwen_provider_test.go:147`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Set
4. Call Header
5. Call WriteHeader
   ... and 5 more steps

---

### TestRankFiles

**ID**: `UT-REP-TestRankFiles`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:568`

**Description**: No description provided

**Test Steps**:
1. Call RankFiles
2. Call Errorf
3. Call Contains
4. Call Contains
5. Call Error

---

### TestRankFilesWithChangedFiles

**ID**: `UT-REP-TestRankFilesWithChangedFiles`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:605`

**Description**: No description provided

**Test Steps**:
1. Call RankFiles
2. Call Error

---

### TestRateLimit_Cleanup

**ID**: `UT-LLM-TestRateLimit_Cleanup`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:405`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call TrackRequest
3. Call CheckBudget
4. Call Error
5. Call Sleep
   ... and 2 more steps

---

### TestRateLimit_ExceedsLimit

**ID**: `UT-LLM-TestRateLimit_ExceedsLimit`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:379`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call TrackRequest
3. Call CheckBudget
4. Call Error
5. Call Contains
   ... and 1 more steps

---

### TestRateLimit_WithinLimit

**ID**: `UT-LLM-TestRateLimit_WithinLimit`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:355`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call CheckBudget
3. Call NoError
4. Call TrackRequest

---

### TestRateLimitedChannel_GetConfig

**ID**: `UT-NOT-TestRateLimitedChannel_GetConfig`

**Package**: `notification`

**File**: `internal/notification/ratelimit_test.go:239`

**Description**: No description provided

**Test Steps**:
1. Call GetConfig
2. Call Equal
3. Call Equal
4. Call NotEmpty

---

### TestRateLimitedChannel_GetName

**ID**: `UT-NOT-TestRateLimitedChannel_GetName`

**Package**: `notification`

**File**: `internal/notification/ratelimit_test.go:213`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call GetName

---

### TestRateLimitedChannel_IsEnabled

**ID**: `UT-NOT-TestRateLimitedChannel_IsEnabled`

**Package**: `notification`

**File**: `internal/notification/ratelimit_test.go:226`

**Description**: No description provided

**Test Steps**:
1. Call True
2. Call IsEnabled

---

### TestRateLimitedChannel_ResetStats

**ID**: `UT-NOT-TestRateLimitedChannel_ResetStats`

**Package**: `notification`

**File**: `internal/notification/ratelimit_test.go:257`

**Description**: No description provided

**Test Steps**:
1. Call Send
2. Call Background
3. Call GetStats
4. Call Equal
5. Call ResetStats
   ... and 2 more steps

---

### TestRateLimitedChannel_Send

**ID**: `UT-NOT-TestRateLimitedChannel_Send`

**Package**: `notification`

**File**: `internal/notification/ratelimit_test.go:148`

**Description**: No description provided

**Test Steps**:
1. Call Send
2. Call Background
3. Call NoError
4. Call Equal
5. Call GetStats
   ... and 3 more steps

---

### TestRateLimitedChannel_SendBlocked

**ID**: `UT-NOT-TestRateLimitedChannel_SendBlocked`

**Package**: `notification`

**File**: `internal/notification/ratelimit_test.go:180`

**Description**: No description provided

**Test Steps**:
1. Call Send
2. Call Background
3. Call NoError
4. Call WithTimeout
5. Call Background
   ... and 5 more steps

---

### TestRateLimiter_Allow

**ID**: `UT-WEB-TestRateLimiter_Allow`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:267`

**Description**: Test 12: Rate limiter - allow check

**Test Steps**:
1. Call SetLimit
2. Call True
3. Call Allow
4. Call True
5. Call Allow
   ... and 2 more steps

---

### TestRateLimiter_Allow

**ID**: `UT-NOT-TestRateLimiter_Allow`

**Package**: `notification`

**File**: `internal/notification/ratelimit_test.go:21`

**Description**: No description provided

**Test Steps**:
1. Call True
2. Call Allow
3. Call True
4. Call Allow
5. Call True
   ... and 3 more steps

---

### TestRateLimiter_Basic

**ID**: `UT-WEB-TestRateLimiter_Basic`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:245`

**Description**: Test 11: Rate limiter - basic functionality

**Test Steps**:
1. Call SetLimit
2. Call Background
3. Call Wait
4. Call NoError
5. Call Now
   ... and 4 more steps

---

### TestRateLimiter_Concurrent

**ID**: `UT-NOT-TestRateLimiter_Concurrent`

**Package**: `notification`

**File**: `internal/notification/ratelimit_test.go:106`

**Description**: No description provided

**Test Steps**:
1. Call Add
2. Call Done
3. Call Allow
4. Call Lock
5. Call Unlock
   ... and 5 more steps

---

### TestRateLimiter_GetAvailableTokens

**ID**: `UT-NOT-TestRateLimiter_GetAvailableTokens`

**Package**: `notification`

**File**: `internal/notification/ratelimit_test.go:49`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call GetAvailableTokens
3. Call Allow
4. Call Equal
5. Call GetAvailableTokens
   ... and 4 more steps

---

### TestRateLimiter_Refill

**ID**: `UT-NOT-TestRateLimiter_Refill`

**Package**: `notification`

**File**: `internal/notification/ratelimit_test.go:33`

**Description**: No description provided

**Test Steps**:
1. Call True
2. Call Allow
3. Call True
4. Call Allow
5. Call False
   ... and 5 more steps

---

### TestRateLimiter_Reset

**ID**: `UT-NOT-TestRateLimiter_Reset`

**Package**: `notification`

**File**: `internal/notification/ratelimit_test.go:62`

**Description**: No description provided

**Test Steps**:
1. Call Allow
2. Call Allow
3. Call Equal
4. Call GetAvailableTokens
5. Call Reset
   ... and 2 more steps

---

### TestRateLimiter_Wait

**ID**: `UT-NOT-TestRateLimiter_Wait`

**Package**: `notification`

**File**: `internal/notification/ratelimit_test.go:75`

**Description**: No description provided

**Test Steps**:
1. Call Allow
2. Call Allow
3. Call Now
4. Call Wait
5. Call Background
   ... and 3 more steps

---

### TestRateLimiter_WaitContextCanceled

**ID**: `UT-NOT-TestRateLimiter_WaitContextCanceled`

**Package**: `notification`

**File**: `internal/notification/ratelimit_test.go:91`

**Description**: No description provided

**Test Steps**:
1. Call Allow
2. Call WithTimeout
3. Call Background
4. Call Wait
5. Call Error

---

### TestRateLimiting

**ID**: `UT-LLM-TestRateLimiting`

**Package**: `llm`

**File**: `internal/llm/provider_features_test.go:189`

**Description**: TestRateLimiting tests request rate limiting

**Test Steps**:
1. Call Background
2. Call CheckBudget
3. Call NoError
4. Call TrackRequest
5. Call New
   ... and 4 more steps

---

### TestReasoningConfigDefaults

**ID**: `UT-LLM-TestReasoningConfigDefaults`

**Package**: `llm`

**File**: `internal/llm/provider_features_test.go:14`

**Description**: TestReasoningConfigDefaults tests default reasoning configuration

**Test Steps**:
1. Call False
2. Call True
3. Call False
4. Call Equal
5. Call Equal
   ... and 1 more steps

---

### TestReasoningCostCalculation

**ID**: `UT-LLM-TestReasoningCostCalculation`

**Package**: `llm`

**File**: `internal/llm/provider_features_test.go:347`

**Description**: TestReasoningCostCalculation tests reasoning cost calculation

**Test Steps**:
1. Call Run
2. Call InDelta
3. Call InDelta
4. Call InDelta

---

### TestReasoningModelDetection

**ID**: `UT-LLM-TestReasoningModelDetection`

**Package**: `llm`

**File**: `internal/llm/provider_features_test.go:26`

**Description**: TestReasoningModelDetection tests automatic reasoning model detection

**Test Steps**:
1. Call Run
2. Call Equal
3. Call Equal

---

### TestReasoningTrace

**ID**: `UT-LLM-TestReasoningTrace`

**Package**: `llm`

**File**: `internal/llm/provider_features_test.go:396`

**Description**: TestReasoningTrace tests reasoning trace extraction

**Test Steps**:
1. Call NotNil
2. Call Len
3. Call Contains
4. Call NotContains
5. Call Contains
   ... and 2 more steps

---

### TestReasoningWorkflow_EndToEnd

**ID**: `UT-LLM-TestReasoningWorkflow_EndToEnd`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:639`

**Description**: Integration tests

**Test Steps**:
1. Call Contains
2. Call NotNil
3. Call Len
4. Call Contains
5. Call Equal
   ... and 3 more steps

---

### TestReasoningWorkflow_MultipleModels

**ID**: `UT-LLM-TestReasoningWorkflow_MultipleModels`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:672`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call True
3. Call Greater
4. Call Greater
5. Call Greater

---

### TestRecommendFormat

**ID**: `UT-EDI-TestRecommendFormat`

**Package**: `editor`

**File**: `internal/editor/model_formats_test.go:241`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Errorf
3. Call Errorf
4. Call Error

---

### TestRecommendFormatReasoningQuality

**ID**: `UT-EDI-TestRecommendFormatReasoningQuality`

**Package**: `editor`

**File**: `internal/editor/model_formats_test.go:391`

**Description**: No description provided

**Test Steps**:
1. Call Error
2. Call Errorf
3. Call Error

---

### TestRefreshCache

**ID**: `UT-REP-TestRefreshCache`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:371`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call RefreshCache
3. Call Fatalf
4. Call Sleep

---

### TestRefreshCacheDisabled

**ID**: `UT-REP-TestRefreshCacheDisabled`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:389`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call RefreshCache
3. Call Error

---

### TestRegisterChannel

**ID**: `UT-NOT-TestRegisterChannel`

**Package**: `notification`

**File**: `internal/notification/engine_test.go:19`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call NoError
3. Call RegisterChannel
4. Call Error

---

### TestRegisterTool

**ID**: `UT-MCP-TestRegisterTool`

**Package**: `mcp`

**File**: `internal/mcp/server_test.go:20`

**Description**: No description provided

**Test Steps**:
1. Call RegisterTool
2. Call NoError
3. Call Equal
4. Call GetToolCount
5. Call RegisterTool
   ... and 3 more steps

---

### TestResetSession

**ID**: `UT-LLM-TestResetSession`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:777`

**Description**: No description provided

**Test Steps**:
1. Call TrackRequest
2. Call GetSessionUsage
3. Call NoError
4. Call ResetSession
5. Call GetSessionUsage
   ... and 3 more steps

---

### TestResilientExecutorCircuitBreakerTrip

**ID**: `UT-AGE-TestResilientExecutorCircuitBreakerTrip`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:403`

**Description**: No description provided

**Test Steps**:
1. Call New
2. Call NewTask
3. Call Execute
4. Call Background
5. Call Error
   ... and 5 more steps

---

### TestResilientExecutorRetry

**ID**: `UT-AGE-TestResilientExecutorRetry`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:368`

**Description**: No description provided

**Test Steps**:
1. Call New
2. Call NewResult
3. Call SetSuccess
4. Call NewTask
5. Call Execute
   ... and 5 more steps

---

### TestResilientExecutorSuccess

**ID**: `UT-AGE-TestResilientExecutorSuccess`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:342`

**Description**: No description provided

**Test Steps**:
1. Call NewTask
2. Call Execute
3. Call Background
4. Call NoError
5. Call NotNil
   ... and 4 more steps

---

### TestResilientExecutorWithNilResult

**ID**: `UT-AGE-TestResilientExecutorWithNilResult`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:595`

**Description**: No description provided

**Test Steps**:
1. Call New
2. Call NewTask
3. Call Execute
4. Call Background
5. Call Error
   ... and 3 more steps

---

### TestRestoreSnapshot

**ID**: `UT-SNA-TestRestoreSnapshot`

**Package**: `snapshots`

**File**: `internal/workflow/snapshots/snapshots_test.go:563`

**Description**: TestRestoreSnapshot tests restoring a snapshot

**Test Steps**:
1. Call Fatalf
2. Call Background
3. Call Command
4. Call Run
5. Call Command
   ... and 5 more steps

---

### TestRestoreSnapshot_DryRun

**ID**: `UT-SNA-TestRestoreSnapshot_DryRun`

**Package**: `snapshots`

**File**: `internal/workflow/snapshots/snapshots_test.go:627`

**Description**: TestRestoreSnapshot_DryRun tests dry run restore

**Test Steps**:
1. Call Fatalf
2. Call Background
3. Call CreateSnapshot
4. Call Fatalf
5. Call RestoreSnapshot
   ... and 3 more steps

---

### TestResultAddArtifact

**ID**: `UT-TAS-TestResultAddArtifact`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:394`

**Description**: No description provided

**Test Steps**:
1. Call Empty
2. Call AddArtifact
3. Call Len
4. Call Equal
5. Call AddArtifact
   ... and 2 more steps

---

### TestResultSetFailure

**ID**: `UT-TAS-TestResultSetFailure`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:372`

**Description**: No description provided

**Test Steps**:
1. Call SetFailure
2. Call False
3. Call Equal
4. Call Error
5. Call Equal

---

### TestResultSetFailureWithNil

**ID**: `UT-TAS-TestResultSetFailureWithNil`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:384`

**Description**: No description provided

**Test Steps**:
1. Call SetFailure
2. Call False
3. Call Empty
4. Call Equal

---

### TestResultSetSuccess

**ID**: `UT-TAS-TestResultSetSuccess`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:355`

**Description**: No description provided

**Test Steps**:
1. Call SetSuccess
2. Call True
3. Call Equal
4. Call Equal
5. Call Empty

---

### TestResultWithComplexOutput

**ID**: `UT-TAS-TestResultWithComplexOutput`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:607`

**Description**: No description provided

**Test Steps**:
1. Call SetSuccess
2. Call True
3. Call Equal
4. Call Equal
5. Call Equal
   ... and 4 more steps

---

### TestRetentionPolicy_OldMessages

**ID**: `UT-COM-TestRetentionPolicy_OldMessages`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:387`

**Description**: Test 12: Retention Policy - Old Normal Messages

**Test Steps**:
1. Call ShouldRetain
2. Call False

---

### TestRetentionPolicy_PinnedMessages

**ID**: `UT-COM-TestRetentionPolicy_PinnedMessages`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:364`

**Description**: Test 10: Retention Policy - Pinned Messages

**Test Steps**:
1. Call ShouldRetain
2. Call True

---

### TestRetentionPolicy_RecentMessages

**ID**: `UT-COM-TestRetentionPolicy_RecentMessages`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:376`

**Description**: Test 11: Retention Policy - Recent Messages

**Test Steps**:
1. Call ShouldRetain
2. Call True

---

### TestRetentionPolicy_SystemMessages

**ID**: `UT-COM-TestRetentionPolicy_SystemMessages`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:353`

**Description**: Test 9: Retention Policy - System Messages

**Test Steps**:
1. Call ShouldRetain
2. Call True

---

### TestRetryBackoffTiming

**ID**: `UT-AGE-TestRetryBackoffTiming`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:548`

**Description**: No description provided

**Test Steps**:
1. Call Now
2. Call Background
3. Call New
4. Call Since
5. Call NoError
   ... and 2 more steps

---

### TestRetryContextCancellation

**ID**: `UT-AGE-TestRetryContextCancellation`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:267`

**Description**: No description provided

**Test Steps**:
1. Call WithTimeout
2. Call Background
3. Call New
4. Call Error
5. Call Equal
   ... and 1 more steps

---

### TestRetryFailureThenSuccess

**ID**: `UT-AGE-TestRetryFailureThenSuccess`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:227`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call New
3. Call NoError
4. Call Equal

---

### TestRetryMaxRetriesExceeded

**ID**: `UT-AGE-TestRetryMaxRetriesExceeded`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:248`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call New
3. Call Error
4. Call Contains
5. Call Error
   ... and 1 more steps

---

### TestRetryNonRetryableError

**ID**: `UT-AGE-TestRetryNonRetryableError`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:290`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call Error
3. Call Equal
4. Call Equal

---

### TestRetryPolicyGetDelay

**ID**: `UT-AGE-TestRetryPolicyGetDelay`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:189`

**Description**: No description provided

**Test Steps**:
1. Call GetDelay
2. Call Equal
3. Call GetDelay
4. Call Equal
5. Call GetDelay
   ... and 3 more steps

---

### TestRetryPolicyShouldRetry

**ID**: `UT-AGE-TestRetryPolicyShouldRetry`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:156`

**Description**: No description provided

**Test Steps**:
1. Call False
2. Call ShouldRetry
3. Call False
4. Call ShouldRetry
5. Call False
   ... and 5 more steps

---

### TestRetryPolicyShouldRetryWithSpecificErrors

**ID**: `UT-AGE-TestRetryPolicyShouldRetryWithSpecificErrors`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:171`

**Description**: No description provided

**Test Steps**:
1. Call New
2. Call New
3. Call True
4. Call ShouldRetry
5. Call False
   ... and 4 more steps

---

### TestRetrySuccess

**ID**: `UT-AGE-TestRetrySuccess`

**Package**: `agent`

**File**: `internal/agent/resilience_test.go:209`

**Description**: No description provided

**Test Steps**:
1. Call Background
2. Call NoError
3. Call Equal

---

### TestRetryableChannel_GetStats

**ID**: `UT-NOT-TestRetryableChannel_GetStats`

**Package**: `notification`

**File**: `internal/notification/retry_test.go:104`

**Description**: No description provided

**Test Steps**:
1. Call GetStats
2. Call Equal

---

### TestRetryableChannel_Send_Success

**ID**: `UT-NOT-TestRetryableChannel_Send_Success`

**Package**: `notification`

**File**: `internal/notification/retry_test.go:58`

**Description**: No description provided

**Test Steps**:
1. Call Send
2. Call Background
3. Call NoError
4. Call GetStats
5. Call Equal
   ... and 3 more steps

---

### TestReviewAgentCollaborate

**ID**: `UT-TYP-TestReviewAgentCollaborate`

**Package**: `types`

**File**: `internal/agent/types/review_agent_test.go:349`

**Description**: TestReviewAgentCollaborate tests collaboration with coding agents

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call NewBaseAgent
5. Call Background
   ... and 5 more steps

---

### TestReviewAgentDefaultReviewType

**ID**: `UT-TYP-TestReviewAgentDefaultReviewType`

**Package**: `types`

**File**: `internal/agent/types/review_agent_test.go:437`

**Description**: TestReviewAgentDefaultReviewType tests default review type

**Test Steps**:
1. Call Contains
2. Call NewToolRegistry
3. Call NoError
4. Call NoError
5. Call Background
   ... and 5 more steps

---

### TestReviewAgentDetermineStaticAnalysisCommands

**ID**: `UT-TYP-TestReviewAgentDetermineStaticAnalysisCommands`

**Package**: `types`

**File**: `internal/agent/types/review_agent_test.go:689`

**Description**: TestReviewAgentDetermineStaticAnalysisCommands tests the determineStaticAnalysisCommands helper

**Test Steps**:
1. Call NoError
2. Call Run
3. Call determineStaticAnalysisCommands
4. Call Contains
5. Call Contains
   ... and 4 more steps

---

### TestReviewAgentExecuteLLMError

**ID**: `UT-TYP-TestReviewAgentExecuteLLMError`

**Package**: `types`

**File**: `internal/agent/types/review_agent_test.go:265`

**Description**: TestReviewAgentExecuteLLMError tests LLM generation error

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call NewTask
   ... and 5 more steps

---

### TestReviewAgentExecuteMissingCodeAndFile

**ID**: `UT-TYP-TestReviewAgentExecuteMissingCodeAndFile`

**Package**: `types`

**File**: `internal/agent/types/review_agent_test.go:231`

**Description**: TestReviewAgentExecuteMissingCodeAndFile tests error when both code and file_path are missing

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call NewTask
   ... and 5 more steps

---

### TestReviewAgentExecutePerformanceReview

**ID**: `UT-TYP-TestReviewAgentExecutePerformanceReview`

**Package**: `types`

**File**: `internal/agent/types/review_agent_test.go:190`

**Description**: TestReviewAgentExecutePerformanceReview tests performance-focused review

**Test Steps**:
1. Call Contains
2. Call NewToolRegistry
3. Call NoError
4. Call NoError
5. Call Background
   ... and 5 more steps

---

### TestReviewAgentExecuteSecurityReview

**ID**: `UT-TYP-TestReviewAgentExecuteSecurityReview`

**Package**: `types`

**File**: `internal/agent/types/review_agent_test.go:149`

**Description**: TestReviewAgentExecuteSecurityReview tests security-focused review

**Test Steps**:
1. Call Contains
2. Call NewToolRegistry
3. Call NoError
4. Call NoError
5. Call Background
   ... and 5 more steps

---

### TestReviewAgentExecuteWithCode

**ID**: `UT-TYP-TestReviewAgentExecuteWithCode`

**Package**: `types`

**File**: `internal/agent/types/review_agent_test.go:108`

**Description**: TestReviewAgentExecuteWithCode tests basic code review with inline code

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call NewTask
   ... and 5 more steps

---

### TestReviewAgentExecuteWithManyIssues

**ID**: `UT-TYP-TestReviewAgentExecuteWithManyIssues`

**Package**: `types`

**File**: `internal/agent/types/review_agent_test.go:301`

**Description**: TestReviewAgentExecuteWithManyIssues tests confidence adjustment for many issues

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call NewTask
   ... and 4 more steps

---

### TestReviewAgentInitialize

**ID**: `UT-TYP-TestReviewAgentInitialize`

**Package**: `types`

**File**: `internal/agent/types/review_agent_test.go:66`

**Description**: TestReviewAgentInitialize tests agent initialization

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call Initialize
   ... and 3 more steps

---

### TestReviewAgentMetrics

**ID**: `UT-TYP-TestReviewAgentMetrics`

**Package**: `types`

**File**: `internal/agent/types/review_agent_test.go:398`

**Description**: TestReviewAgentMetrics tests metrics recording

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call NewTask
   ... and 5 more steps

---

### TestReviewAgentReadFile

**ID**: `UT-TYP-TestReviewAgentReadFile`

**Package**: `types`

**File**: `internal/agent/types/review_agent_test.go:478`

**Description**: TestReviewAgentReadFile tests the readFile helper function

**Test Steps**:
1. Call Run
2. Call NoError
3. Call Background
4. Call readFile
5. Call NoError
   ... and 5 more steps

---

### TestReviewAgentRunStaticAnalysis

**ID**: `UT-TYP-TestReviewAgentRunStaticAnalysis`

**Package**: `types`

**File**: `internal/agent/types/review_agent_test.go:574`

**Description**: TestReviewAgentRunStaticAnalysis tests the runStaticAnalysis helper function

**Test Steps**:
1. Call Run
2. Call NoError
3. Call Background
4. Call runStaticAnalysis
5. Call NoError
   ... and 5 more steps

---

### TestReviewAgentShutdown

**ID**: `UT-TYP-TestReviewAgentShutdown`

**Package**: `types`

**File**: `internal/agent/types/review_agent_test.go:87`

**Description**: TestReviewAgentShutdown tests agent shutdown

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call Shutdown
   ... and 3 more steps

---

### TestSSHWorkerPool_AddWorker

**ID**: `UT-WOR-TestSSHWorkerPool_AddWorker`

**Package**: `worker`

**File**: `internal/worker/ssh_pool_test.go:22`

**Description**: TestSSHWorkerPool_AddWorker tests adding workers to the pool

**Test Steps**:
1. Call Background
2. Call AddWorker
3. Call Error
4. Call Contains
5. Call Error

---

### TestSSHWorkerPool_ConcurrentAccess

**ID**: `UT-WOR-TestSSHWorkerPool_ConcurrentAccess`

**Package**: `worker`

**File**: `internal/worker/ssh_pool_test.go:281`

**Description**: TestSSHWorkerPool_ConcurrentAccess tests concurrent access safety

**Test Steps**:
1. Call Background
2. Call Lock
3. Call New
4. Call Unlock
5. Call GetWorkerStats
   ... and 4 more steps

---

### TestSSHWorkerPool_Creation

**ID**: `UT-WOR-TestSSHWorkerPool_Creation`

**Package**: `worker`

**File**: `internal/worker/ssh_pool_test.go:12`

**Description**: TestSSHWorkerPool_Creation tests SSH worker pool creation

**Test Steps**:
1. Call NotNil
2. Call True
3. Call False

---

### TestSSHWorkerPool_DetectWorkerCapabilities

**ID**: `UT-WOR-TestSSHWorkerPool_DetectWorkerCapabilities`

**Package**: `worker`

**File**: `internal/worker/ssh_pool_test.go:256`

**Description**: TestSSHWorkerPool_DetectWorkerCapabilities tests capability detection

**Test Steps**:
1. Call Background
2. Call New
3. Call detectWorkerCapabilities
4. Call Error
5. Call Contains
   ... and 5 more steps

---

### TestSSHWorkerPool_ErrorHandling

**ID**: `UT-WOR-TestSSHWorkerPool_ErrorHandling`

**Package**: `worker`

**File**: `internal/worker/ssh_pool_test.go:321`

**Description**: TestSSHWorkerPool_ErrorHandling tests various error scenarios

**Test Steps**:
1. Call Background
2. Call AddWorker
3. Call Error
4. Call AddWorker
5. Call Error
   ... and 2 more steps

---

### TestSSHWorkerPool_ExecuteCommand

**ID**: `UT-WOR-TestSSHWorkerPool_ExecuteCommand`

**Package**: `worker`

**File**: `internal/worker/ssh_pool_test.go:150`

**Description**: TestSSHWorkerPool_ExecuteCommand tests command execution

**Test Steps**:
1. Call Background
2. Call New
3. Call ExecuteCommand
4. Call Error
5. Call Contains
   ... and 5 more steps

---

### TestSSHWorkerPool_GetWorkerStats

**ID**: `UT-WOR-TestSSHWorkerPool_GetWorkerStats`

**Package**: `worker`

**File**: `internal/worker/ssh_pool_test.go:107`

**Description**: TestSSHWorkerPool_GetWorkerStats tests statistics collection

**Test Steps**:
1. Call Background
2. Call New
3. Call New
4. Call GetWorkerStats
5. Call Equal
   ... and 5 more steps

---

### TestSSHWorkerPool_HealthCheck

**ID**: `UT-WOR-TestSSHWorkerPool_HealthCheck`

**Package**: `worker`

**File**: `internal/worker/ssh_pool_test.go:67`

**Description**: TestSSHWorkerPool_HealthCheck tests health checking

**Test Steps**:
1. Call Background
2. Call New
3. Call New
4. Call HealthCheck
5. Call NoError
   ... and 2 more steps

---

### TestSSHWorkerPool_InstallHelixCLI

**ID**: `UT-WOR-TestSSHWorkerPool_InstallHelixCLI`

**Package**: `worker`

**File**: `internal/worker/ssh_pool_test.go:236`

**Description**: TestSSHWorkerPool_InstallHelixCLI tests auto-installation

**Test Steps**:
1. Call Background
2. Call New
3. Call installHelixCLI
4. Call Error
5. Call Contains
   ... and 1 more steps

---

### TestSSHWorkerPool_RemoveWorker

**ID**: `UT-WOR-TestSSHWorkerPool_RemoveWorker`

**Package**: `worker`

**File**: `internal/worker/ssh_pool_test.go:44`

**Description**: TestSSHWorkerPool_RemoveWorker tests worker removal

**Test Steps**:
1. Call Background
2. Call New
3. Call RemoveWorker
4. Call NoError
5. Call NotContains
   ... and 5 more steps

---

### TestSSHWorkerPool_ResourceManagement

**ID**: `UT-WOR-TestSSHWorkerPool_ResourceManagement`

**Package**: `worker`

**File**: `internal/worker/ssh_pool_test.go:361`

**Description**: TestSSHWorkerPool_ResourceManagement tests resource management

**Test Steps**:
1. Call Background
2. Call New
3. Call GetWorkerStats
4. Call Equal
5. Call Equal
   ... and 4 more steps

---

### TestSSHWorkerPool_ValidateSSHConfig

**ID**: `UT-WOR-TestSSHWorkerPool_ValidateSSHConfig`

**Package**: `worker`

**File**: `internal/worker/ssh_pool_test.go:179`

**Description**: TestSSHWorkerPool_ValidateSSHConfig tests SSH configuration validation

**Test Steps**:
1. Call validateSSHConfig
2. Call NoError
3. Call Run
4. Call validateSSHConfig
5. Call Error
   ... and 2 more steps

---

### TestSSHWorkerStats_String

**ID**: `UT-WOR-TestSSHWorkerStats_String`

**Package**: `worker`

**File**: `internal/worker/ssh_pool_test.go:341`

**Description**: TestSSHWorkerStats_String tests statistics string representation

**Test Steps**:
1. Call Equal
2. Call Equal
3. Call Equal
4. Call Equal
5. Call Equal
   ... and 1 more steps

---

### TestSaveColorScheme

**ID**: `UT-LOG-TestSaveColorScheme`

**Package**: `logo`

**File**: `internal/logo/processor_test.go:51`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call NoError
3. Call RemoveAll
4. Call Join
5. Call MkdirAll
   ... and 5 more steps

---

### TestScreenshot

**ID**: `UT-BRO-TestScreenshot`

**Package**: `browser`

**File**: `internal/tools/browser/browser_test.go:278`

**Description**: TestScreenshot tests screenshot capture

**Test Steps**:
1. Call WithTimeout
2. Call Background
3. Call Launch
4. Call Skipf
5. Call Close
   ... and 5 more steps

---

### TestSearchReplaceEditorApply

**ID**: `UT-EDI-TestSearchReplaceEditorApply`

**Package**: `editor`

**File**: `internal/editor/search_replace_editor_test.go:10`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Run
5. Call Join
   ... and 5 more steps

---

### TestSearchReplaceEditorApplyToLines

**ID**: `UT-EDI-TestSearchReplaceEditorApplyToLines`

**Package**: `editor`

**File**: `internal/editor/search_replace_editor_test.go:157`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Join
5. Call WriteFile
   ... and 5 more steps

---

### TestSearchReplaceEditorCountMatches

**ID**: `UT-EDI-TestSearchReplaceEditorCountMatches`

**Package**: `editor`

**File**: `internal/editor/search_replace_editor_test.go:241`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call CountMatches
3. Call Error
4. Call Errorf
5. Call Errorf

---

### TestSearchReplaceEditorGetStats

**ID**: `UT-EDI-TestSearchReplaceEditorGetStats`

**Package**: `editor`

**File**: `internal/editor/search_replace_editor_test.go:305`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call GetStats
3. Call Fatalf
4. Call Errorf
5. Call Errorf
   ... and 1 more steps

---

### TestSearchReplaceEditorLargeFile

**ID**: `UT-EDI-TestSearchReplaceEditorLargeFile`

**Package**: `editor`

**File**: `internal/editor/search_replace_editor_test.go:369`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Join
5. Call Repeat
   ... and 5 more steps

---

### TestSearchReplaceEditorNoOperations

**ID**: `UT-EDI-TestSearchReplaceEditorNoOperations`

**Package**: `editor`

**File**: `internal/editor/search_replace_editor_test.go:495`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Join
5. Call WriteFile
   ... and 3 more steps

---

### TestSearchReplaceEditorRegexEdgeCases

**ID**: `UT-EDI-TestSearchReplaceEditorRegexEdgeCases`

**Package**: `editor`

**File**: `internal/editor/search_replace_editor_test.go:416`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Run
5. Call Join
   ... and 5 more steps

---

### TestSearchReplaceEditorValidateOperation

**ID**: `UT-EDI-TestSearchReplaceEditorValidateOperation`

**Package**: `editor`

**File**: `internal/editor/search_replace_editor_test.go:195`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call ValidateOperation
3. Call Error
4. Call Errorf

---

### TestSecurityMiddleware

**ID**: `UT-SER-TestSecurityMiddleware`

**Package**: `server`

**File**: `internal/server/server_test.go:100`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call New
3. Call Use
4. Call GET
5. Call JSON
   ... and 5 more steps

---

### TestSelectBestFormat

**ID**: `UT-EDI-TestSelectBestFormat`

**Package**: `editor`

**File**: `internal/editor/model_formats_test.go:140`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Errorf

---

### TestSelectBestFormatEdgeCases

**ID**: `UT-EDI-TestSelectBestFormatEdgeCases`

**Package**: `editor`

**File**: `internal/editor/model_formats_test.go:369`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Error

---

### TestSelectFormatByComplexity

**ID**: `UT-EDI-TestSelectFormatByComplexity`

**Package**: `editor`

**File**: `internal/editor/model_formats_test.go:204`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Errorf

---

### TestSelectFormatCaseInsensitivity

**ID**: `UT-EDI-TestSelectFormatCaseInsensitivity`

**Package**: `editor`

**File**: `internal/editor/model_formats_test.go:346`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Errorf

---

### TestSelectFormatForModel

**ID**: `UT-EDI-TestSelectFormatForModel`

**Package**: `editor`

**File**: `internal/editor/model_formats_test.go:7`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Errorf

---

### TestSemanticSummarizationStrategy_Execute

**ID**: `UT-COM-TestSemanticSummarizationStrategy_Execute`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:268`

**Description**: Test 6: Semantic Summarization Strategy

**Test Steps**:
1. Call Execute
2. Call Background
3. Call NoError
4. Call Less
5. Call Greater
   ... and 2 more steps

---

### TestSemanticSummarizationStrategy_PreserveTypes

**ID**: `UT-COM-TestSemanticSummarizationStrategy_PreserveTypes`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:291`

**Description**: Test 7: Semantic Summarization with Preserved Types

**Test Steps**:
1. Call Execute
2. Call Background
3. Call NoError
4. Call True
5. Call True

---

### TestSendDirect

**ID**: `UT-NOT-TestSendDirect`

**Package**: `notification`

**File**: `internal/notification/engine_test.go:62`

**Description**: No description provided

**Test Steps**:
1. Call RegisterChannel
2. Call SendDirect
3. Call Background
4. Call NoError
5. Call NotEqual
   ... and 3 more steps

---

### TestSensitiveFileDetector

**ID**: `UT-FIL-TestSensitiveFileDetector`

**Package**: `filesystem`

**File**: `internal/tools/filesystem/filesystem_test.go:710`

**Description**: TestSensitiveFileDetector tests sensitive file detection

**Test Steps**:
1. Call Run
2. Call IsSensitive
3. Call Errorf

---

### TestServerInitialization

**ID**: `UT-SER-TestServerInitialization`

**Package**: `server`

**File**: `internal/server/server_test.go:148`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call NotNil
3. Call Routes
4. Call Greater

---

### TestServerPortConfiguration

**ID**: `UT-SER-TestServerPortConfiguration`

**Package**: `server`

**File**: `internal/server/server_test.go:181`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Equal
3. Call NotNil

---

### TestServerWithNilDependencies

**ID**: `UT-SER-TestServerWithNilDependencies`

**Package**: `server`

**File**: `internal/server/server_test.go:162`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call NotNil
3. Call Equal

---

### TestSessionCleanup

**ID**: `UT-LLM-TestSessionCleanup`

**Package**: `llm`

**File**: `internal/llm/provider_features_test.go:253`

**Description**: TestSessionCleanup tests old session cleanup

**Test Steps**:
1. Call String
2. Call New
3. Call TrackRequest
4. Call New
5. Call Sleep
   ... and 4 more steps

---

### TestSessionJSONTags

**ID**: `UT-SES-TestSessionJSONTags`

**Package**: `session`

**File**: `internal/session/session_test.go:47`

**Description**: No description provided

**Test Steps**:
1. Call Now
2. Call Now
3. Call NotEmpty
4. Call NotEmpty
5. Call NotEmpty
   ... and 3 more steps

---

### TestSessionStruct

**ID**: `UT-SES-TestSessionStruct`

**Package**: `session`

**File**: `internal/session/session_test.go:10`

**Description**: No description provided

**Test Steps**:
1. Call Now
2. Call Equal
3. Call Equal
4. Call Equal
5. Call Equal
   ... and 4 more steps

---

### TestSetActiveProject

**ID**: `UT-PRO-TestSetActiveProject`

**Package**: `project`

**File**: `internal/project/manager_test.go:95`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call NoError
3. Call RemoveAll
4. Call CreateProject
5. Call Background
   ... and 5 more steps

---

### TestSignalHandling

**ID**: `UT-SHE-TestSignalHandling`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:263`

**Description**: TestSignalHandling tests signal handling

**Test Steps**:
1. Call Skip
2. Call ExecuteAsync
3. Call Background
4. Call NoError
5. Call Sleep
   ... and 3 more steps

---

### TestSilenceDetector

**ID**: `UT-VOI-TestSilenceDetector`

**Package**: `voice`

**File**: `internal/tools/voice/voice_test.go:249`

**Description**: TestSilenceDetector tests silence detection

**Test Steps**:
1. Call Now
2. Call IsSilent
3. Call Error
4. Call Now
5. Call IsSilent
   ... and 5 more steps

---

### TestSimpleCommandExecution

**ID**: `UT-SHE-TestSimpleCommandExecution`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:18`

**Description**: TestSimpleCommandExecution tests basic command execution

**Test Steps**:
1. Call Execute
2. Call Background
3. Call NoError
4. Call Equal
5. Call Contains
   ... and 1 more steps

---

### TestSlackChannel_GetConfig

**ID**: `UT-NOT-TestSlackChannel_GetConfig`

**Package**: `notification`

**File**: `internal/notification/slack_test.go:185`

**Description**: No description provided

**Test Steps**:
1. Call GetConfig
2. Call Equal
3. Call Equal
4. Call Equal

---

### TestSlackChannel_GetIconForType

**ID**: `UT-NOT-TestSlackChannel_GetIconForType`

**Package**: `notification`

**File**: `internal/notification/slack_test.go:163`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call getIconForType
3. Call Equal

---

### TestSlackChannel_Send

**ID**: `UT-NOT-TestSlackChannel_Send`

**Package**: `notification`

**File**: `internal/notification/slack_test.go:48`

**Description**: No description provided

**Test Steps**:
1. Call ReadAll
2. Call Unmarshal
3. Call NoError
4. Call Equal
5. Call Contains
   ... and 5 more steps

---

### TestSlackChannel_Send_Disabled

**ID**: `UT-NOT-TestSlackChannel_Send_Disabled`

**Package**: `notification`

**File**: `internal/notification/slack_test.go:149`

**Description**: No description provided

**Test Steps**:
1. Call Send
2. Call Background
3. Call Error
4. Call Contains
5. Call Error

---

### TestSlidingWindowStrategy_Estimate

**ID**: `UT-COM-TestSlidingWindowStrategy_Estimate`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:249`

**Description**: Test 5: Sliding Window Strategy Estimate

**Test Steps**:
1. Call Estimate
2. Call NoError
3. Call Greater
4. Call Greater
5. Call Equal
   ... and 1 more steps

---

### TestSlidingWindowStrategy_Execute

**ID**: `UT-COM-TestSlidingWindowStrategy_Execute`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:203`

**Description**: Test 3: Sliding Window Strategy Basic

**Test Steps**:
1. Call Execute
2. Call Background
3. Call NoError
4. Call Less
5. Call Equal
   ... and 2 more steps

---

### TestSlidingWindowStrategy_WithPinnedMessages

**ID**: `UT-COM-TestSlidingWindowStrategy_WithPinnedMessages`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:222`

**Description**: Test 4: Sliding Window Strategy with Pinned Messages

**Test Steps**:
1. Call Execute
2. Call Background
3. Call NoError
4. Call Equal

---

### TestStateManager

**ID**: `UT-PLA-TestStateManager`

**Package**: `planmode`

**File**: `internal/workflow/planmode/planmode_test.go:371`

**Description**: Test State Manager

**Test Steps**:
1. Call Run
2. Call StorePlan
3. Call NoError
4. Call GetPlan
5. Call NoError
   ... and 5 more steps

---

### TestStatusConstants

**ID**: `UT-SES-TestStatusConstants`

**Package**: `session`

**File**: `internal/session/session_test.go:40`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call Equal
3. Call Equal
4. Call Equal

---

### TestStderrCapture

**ID**: `UT-SHE-TestStderrCapture`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:317`

**Description**: TestStderrCapture tests stderr capture

**Test Steps**:
1. Call Execute
2. Call Background
3. Call NoError
4. Call Equal
5. Call Contains

---

### TestStreamingExecution

**ID**: `UT-SHE-TestStreamingExecution`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:188`

**Description**: TestStreamingExecution tests real-time output streaming

**Test Steps**:
1. Call ExecuteStream
2. Call Background
3. Call NoError
4. Call Equal
5. Call Equal

---

### TestStreamingWithErrors

**ID**: `UT-SHE-TestStreamingWithErrors`

**Package**: `shell`

**File**: `internal/tools/shell/shell_test.go:626`

**Description**: TestStreamingWithErrors tests streaming execution with errors

**Test Steps**:
1. Call ExecuteStream
2. Call Background
3. Call NoError
4. Call Equal
5. Call True
   ... and 3 more steps

---

### TestSupportsFormat

**ID**: `UT-EDI-TestSupportsFormat`

**Package**: `editor`

**File**: `internal/editor/model_formats_test.go:117`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Errorf

---

### TestSwitchHistory

**ID**: `UT-VIS-TestSwitchHistory`

**Package**: `vision`

**File**: `internal/llm/vision/vision_test.go:483`

**Description**: TestSwitchHistory tests switch event tracking

**Test Steps**:
1. Call Background
2. Call Switch
3. Call Fatalf
4. Call Error
5. Call GetHistory
   ... and 5 more steps

---

### TestSwitchModeValidation

**ID**: `UT-VIS-TestSwitchModeValidation`

**Package**: `vision`

**File**: `internal/llm/vision/vision_test.go:642`

**Description**: TestSwitchModeValidation tests switch mode validation

**Test Steps**:
1. Call Run
2. Call IsValid
3. Call Errorf

---

### TestSwitchModes

**ID**: `UT-VIS-TestSwitchModes`

**Package**: `vision`

**File**: `internal/llm/vision/vision_test.go:412`

**Description**: TestSwitchModes tests different switch modes

**Test Steps**:
1. Call Run
2. Call Fatalf
3. Call Background
4. Call SetCurrentModel
5. Call ProcessInput
   ... and 5 more steps

---

### TestSymbolTypes

**ID**: `UT-REP-TestSymbolTypes`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:532`

**Description**: No description provided

**Test Steps**:
1. Call Errorf

---

### TestTaskBlock

**ID**: `UT-TAS-TestTaskBlock`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:149`

**Description**: No description provided

**Test Steps**:
1. Call Block
2. Call Equal
3. Call Equal
4. Call NotNil
5. Call Equal
   ... and 2 more steps

---

### TestTaskBlockUnblockLifecycle

**ID**: `UT-TAS-TestTaskBlockUnblockLifecycle`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:487`

**Description**: No description provided

**Test Steps**:
1. Call Block
2. Call Equal
3. Call Contains
4. Call False
5. Call CanStart
   ... and 5 more steps

---

### TestTaskCanStart

**ID**: `UT-TAS-TestTaskCanStart`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:279`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call CanStart
3. Call Equal

---

### TestTaskCancel

**ID**: `UT-TAS-TestTaskCancel`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:190`

**Description**: No description provided

**Test Steps**:
1. Call Start
2. Call Cancel
3. Call Equal
4. Call NotNil
5. Call Equal
   ... and 2 more steps

---

### TestTaskCancelFromDifferentStates

**ID**: `UT-TAS-TestTaskCancelFromDifferentStates`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:580`

**Description**: No description provided

**Test Steps**:
1. Call Cancel
2. Call Equal
3. Call Equal

---

### TestTaskComplete

**ID**: `UT-TAS-TestTaskComplete`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:92`

**Description**: No description provided

**Test Steps**:
1. Call Start
2. Call Complete
3. Call Equal
4. Call NotNil
5. Call Equal
   ... and 4 more steps

---

### TestTaskCompleteWithoutStart

**ID**: `UT-TAS-TestTaskCompleteWithoutStart`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:110`

**Description**: No description provided

**Test Steps**:
1. Call Complete
2. Call Equal
3. Call NotNil
4. Call Equal
5. Call Equal
   ... and 1 more steps

---

### TestTaskDurationCalculation

**ID**: `UT-TAS-TestTaskDurationCalculation`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:635`

**Description**: No description provided

**Test Steps**:
1. Call Start
2. Call Sleep
3. Call Complete
4. Call Greater
5. Call Duration
   ... and 2 more steps

---

### TestTaskFail

**ID**: `UT-TAS-TestTaskFail`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:123`

**Description**: No description provided

**Test Steps**:
1. Call Start
2. Call Fail
3. Call Equal
4. Call NotNil
5. Call Equal
   ... and 3 more steps

---

### TestTaskFailWithNilMetadata

**ID**: `UT-TAS-TestTaskFailWithNilMetadata`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:137`

**Description**: No description provided

**Test Steps**:
1. Call Fail
2. Call Equal
3. Call NotNil
4. Call Equal

---

### TestTaskFailureLifecycle

**ID**: `UT-TAS-TestTaskFailureLifecycle`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:472`

**Description**: No description provided

**Test Steps**:
1. Call Start
2. Call True
3. Call IsActive
4. Call Fail
5. Call False
   ... and 5 more steps

---

### TestTaskIsActive

**ID**: `UT-TAS-TestTaskIsActive`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:324`

**Description**: No description provided

**Test Steps**:
1. Call False
2. Call IsActive
3. Call Start
4. Call True
5. Call IsActive
   ... and 3 more steps

---

### TestTaskIsCompleted

**ID**: `UT-TAS-TestTaskIsCompleted`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:306`

**Description**: No description provided

**Test Steps**:
1. Call False
2. Call IsCompleted
3. Call Complete
4. Call True
5. Call IsCompleted

---

### TestTaskIsFailed

**ID**: `UT-TAS-TestTaskIsFailed`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:315`

**Description**: No description provided

**Test Steps**:
1. Call False
2. Call IsFailed
3. Call Fail
4. Call True
5. Call IsFailed

---

### TestTaskIsReady

**ID**: `UT-TAS-TestTaskIsReady`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:207`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call IsReady
3. Call Equal

---

### TestTaskLifecycle

**ID**: `UT-TAS-TestTaskLifecycle`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:448`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call False
3. Call IsActive
4. Call False
5. Call IsCompleted
   ... and 5 more steps

---

### TestTaskManager_CompleteTask

**ID**: `UT-TAS-TestTaskManager_CompleteTask`

**Package**: `task`

**File**: `internal/task/manager_test.go:59`

**Description**: No description provided

**Test Steps**:
1. Call CreateTask
2. Call Fatalf
3. Call CompleteTask
4. Call Fatalf

---

### TestTaskManager_CreateTask

**ID**: `UT-TAS-TestTaskManager_CreateTask`

**Package**: `task`

**File**: `internal/task/manager_test.go:25`

**Description**: No description provided

**Test Steps**:
1. Call CreateTask
2. Call Fatalf
3. Call Error
4. Call Errorf
5. Call Errorf
   ... and 1 more steps

---

### TestTaskManager_FailTask

**ID**: `UT-TAS-TestTaskManager_FailTask`

**Package**: `task`

**File**: `internal/task/manager_test.go:90`

**Description**: No description provided

**Test Steps**:
1. Call CreateTask
2. Call Fatalf
3. Call FailTask
4. Call Fatalf

---

### TestTaskManager_GetTaskProgress

**ID**: `UT-TAS-TestTaskManager_GetTaskProgress`

**Package**: `task`

**File**: `internal/task/manager_test.go:224`

**Description**: No description provided

**Test Steps**:
1. Call CreateTask
2. Call Fatalf
3. Call GetTaskProgress
4. Call Fatalf
5. Call Errorf
   ... and 2 more steps

---

### TestTaskMetrics

**ID**: `UT-TAS-TestTaskMetrics`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:507`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call Equal
3. Call Equal
4. Call Equal
5. Call Equal
   ... and 3 more steps

---

### TestTaskMultipleStateTransitions

**ID**: `UT-TAS-TestTaskMultipleStateTransitions`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:563`

**Description**: No description provided

**Test Steps**:
1. Call Block
2. Call Equal
3. Call Unblock
4. Call Equal
5. Call Start
   ... and 3 more steps

---

### TestTaskPriority

**ID**: `UT-WOR-TestTaskPriority`

**Package**: `worker`

**File**: `internal/worker/distributed_manager_test.go:121`

**Description**: TestTaskPriority tests task priority handling

**Test Steps**:
1. Call Skip
2. Call Errorf
3. Call Logf
4. Call Log

---

### TestTaskQueue_AddAndGet

**ID**: `UT-TAS-TestTaskQueue_AddAndGet`

**Package**: `task`

**File**: `internal/task/manager_test.go:115`

**Description**: No description provided

**Test Steps**:
1. Call New
2. Call Now
3. Call Now
4. Call New
5. Call Now
   ... and 5 more steps

---

### TestTaskQueue_Stats

**ID**: `UT-TAS-TestTaskQueue_Stats`

**Package**: `task`

**File**: `internal/task/manager_test.go:188`

**Description**: No description provided

**Test Steps**:
1. Call AddTask
2. Call New
3. Call AddTask
4. Call New
5. Call AddTask
   ... and 5 more steps

---

### TestTaskStart

**ID**: `UT-TAS-TestTaskStart`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:74`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call Nil
3. Call Empty
4. Call Start
5. Call Equal
   ... and 4 more steps

---

### TestTaskStatusTransitions

**ID**: `UT-WOR-TestTaskStatusTransitions`

**Package**: `worker`

**File**: `internal/worker/distributed_manager_test.go:210`

**Description**: TestTaskStatusTransitions tests task status transitions

**Test Steps**:
1. Call Now
2. Call Error
3. Call Error
4. Call Error
5. Call Error
   ... and 1 more steps

---

### TestTaskUnblock

**ID**: `UT-TAS-TestTaskUnblock`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:164`

**Description**: No description provided

**Test Steps**:
1. Call Block
2. Call Equal
3. Call NotEmpty
4. Call Unblock
5. Call Equal
   ... and 3 more steps

---

### TestTaskUnblockWhenNotBlocked

**ID**: `UT-TAS-TestTaskUnblockWhenNotBlocked`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:180`

**Description**: No description provided

**Test Steps**:
1. Call Unblock
2. Call Equal

---

### TestTaskWithDependencies

**ID**: `UT-TAS-TestTaskWithDependencies`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:427`

**Description**: No description provided

**Test Steps**:
1. Call False
2. Call IsReady
3. Call False
4. Call IsReady
5. Call True
   ... and 1 more steps

---

### TestTaskWithEmptyDependencies

**ID**: `UT-TAS-TestTaskWithEmptyDependencies`

**Package**: `task`

**File**: `internal/agent/task/task_test.go:599`

**Description**: No description provided

**Test Steps**:
1. Call True
2. Call IsReady
3. Call True
4. Call IsReady

---

### TestTelegramChannel

**ID**: `UT-NOT-TestTelegramChannel`

**Package**: `notification`

**File**: `internal/notification/engine_test.go:103`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call Equal
3. Call GetName
4. Call True
5. Call IsEnabled
   ... and 3 more steps

---

### TestTelegramChannel_GetConfig

**ID**: `UT-NOT-TestTelegramChannel_GetConfig`

**Package**: `notification`

**File**: `internal/notification/telegram_test.go:191`

**Description**: No description provided

**Test Steps**:
1. Call GetConfig
2. Call Contains
3. Call Contains
4. Call Equal

---

### TestTelegramChannel_MaskToken

**ID**: `UT-NOT-TestTelegramChannel_MaskToken`

**Package**: `notification`

**File**: `internal/notification/telegram_test.go:203`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call maskToken
3. Call Equal

---

### TestTelegramChannel_Send

**ID**: `UT-NOT-TestTelegramChannel_Send`

**Package**: `notification`

**File**: `internal/notification/telegram_test.go:58`

**Description**: No description provided

**Test Steps**:
1. Call ReadAll
2. Call Unmarshal
3. Call NoError
4. Call Equal
5. Call Equal
   ... and 5 more steps

---

### TestTelegramChannel_Send_Disabled

**ID**: `UT-NOT-TestTelegramChannel_Send_Disabled`

**Package**: `notification`

**File**: `internal/notification/telegram_test.go:177`

**Description**: No description provided

**Test Steps**:
1. Call Send
2. Call Background
3. Call Error
4. Call Contains
5. Call Error

---

### TestTestingAgentCollaborate

**ID**: `UT-TYP-TestTestingAgentCollaborate`

**Package**: `types`

**File**: `internal/agent/types/testing_agent_test.go:296`

**Description**: TestTestingAgentCollaborate tests collaboration with coding agents

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call NewBaseAgent
5. Call Background
   ... and 3 more steps

---

### TestTestingAgentExecuteGenerate

**ID**: `UT-TYP-TestTestingAgentExecuteGenerate`

**Package**: `types`

**File**: `internal/agent/types/testing_agent_test.go:108`

**Description**: TestTestingAgentExecuteGenerate tests test generation without execution

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call NewTask
   ... and 4 more steps

---

### TestTestingAgentExecuteLLMError

**ID**: `UT-TYP-TestTestingAgentExecuteLLMError`

**Package**: `types`

**File**: `internal/agent/types/testing_agent_test.go:260`

**Description**: TestTestingAgentExecuteLLMError tests LLM generation error

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call NewTask
   ... and 5 more steps

---

### TestTestingAgentExecuteMissingCode

**ID**: `UT-TYP-TestTestingAgentExecuteMissingCode`

**Package**: `types`

**File**: `internal/agent/types/testing_agent_test.go:226`

**Description**: TestTestingAgentExecuteMissingCode tests error when code is missing

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call NewTask
   ... and 5 more steps

---

### TestTestingAgentExecuteTests

**ID**: `UT-TYP-TestTestingAgentExecuteTests`

**Package**: `types`

**File**: `internal/agent/types/testing_agent_test.go:455`

**Description**: TestTestingAgentExecuteTests tests the executeTests helper function

**Test Steps**:
1. Call Run
2. Call NoError
3. Call Background
4. Call executeTests
5. Call NoError
   ... and 5 more steps

---

### TestTestingAgentExecuteWithFilePath

**ID**: `UT-TYP-TestTestingAgentExecuteWithFilePath`

**Package**: `types`

**File**: `internal/agent/types/testing_agent_test.go:146`

**Description**: TestTestingAgentExecuteWithFilePath tests test generation with file path

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call NewTask
   ... and 4 more steps

---

### TestTestingAgentExecuteWithFramework

**ID**: `UT-TYP-TestTestingAgentExecuteWithFramework`

**Package**: `types`

**File**: `internal/agent/types/testing_agent_test.go:185`

**Description**: TestTestingAgentExecuteWithFramework tests test generation with custom framework

**Test Steps**:
1. Call Contains
2. Call NewToolRegistry
3. Call NoError
4. Call NoError
5. Call Background
   ... and 5 more steps

---

### TestTestingAgentInitialize

**ID**: `UT-TYP-TestTestingAgentInitialize`

**Package**: `types`

**File**: `internal/agent/types/testing_agent_test.go:66`

**Description**: TestTestingAgentInitialize tests agent initialization

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call Initialize
   ... and 3 more steps

---

### TestTestingAgentShutdown

**ID**: `UT-TYP-TestTestingAgentShutdown`

**Package**: `types`

**File**: `internal/agent/types/testing_agent_test.go:87`

**Description**: TestTestingAgentShutdown tests agent shutdown

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call Shutdown
   ... and 3 more steps

---

### TestTestingAgentTaskMetrics

**ID**: `UT-TYP-TestTestingAgentTaskMetrics`

**Package**: `types`

**File**: `internal/agent/types/testing_agent_test.go:343`

**Description**: TestTestingAgentTaskMetrics tests task metrics recording

**Test Steps**:
1. Call NewToolRegistry
2. Call NoError
3. Call NoError
4. Call Background
5. Call NewTask
   ... and 4 more steps

---

### TestTokenBudgetEnforcement

**ID**: `UT-LLM-TestTokenBudgetEnforcement`

**Package**: `llm`

**File**: `internal/llm/provider_features_test.go:109`

**Description**: TestTokenBudgetEnforcement tests token budget enforcement

**Test Steps**:
1. Call Background
2. Call CheckBudget
3. Call Error
4. Call Contains
5. Call Error
   ... and 5 more steps

---

### TestTokenBudgetWarnings

**ID**: `UT-LLM-TestTokenBudgetWarnings`

**Package**: `llm`

**File**: `internal/llm/provider_features_test.go:163`

**Description**: TestTokenBudgetWarnings tests warning thresholds

**Test Steps**:
1. Call Background
2. Call New
3. Call TrackRequest
4. Call CheckBudget
5. Call Error
   ... and 4 more steps

---

### TestTokenCache

**ID**: `UT-COM-TestTokenCache`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:550`

**Description**: Test 21: Token Cache

**Test Steps**:
1. Call Set
2. Call Get
3. Call True
4. Call Equal
5. Call Get
   ... and 5 more steps

---

### TestTokenCounter

**ID**: `UT-MAP-TestTokenCounter`

**Package**: `mapping`

**File**: `internal/tools/mapping/mapping_test.go:339`

**Description**: TestTokenCounter tests token counting

**Test Steps**:
1. Call Run
2. Call Count
3. Call Greater

---

### TestTokenCounter_Count

**ID**: `UT-COM-TestTokenCounter_Count`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:151`

**Description**: Test 1: Token Counter Basic Functionality

**Test Steps**:
1. Call Run
2. Call Count
3. Call GreaterOrEqual
4. Call Count
5. Call Equal

---

### TestTokenCounter_CountConversation

**ID**: `UT-COM-TestTokenCounter_CountConversation`

**Package**: `compression`

**File**: `internal/llm/compression/compression_test.go:189`

**Description**: Test 2: Token Counter Conversation Counting

**Test Steps**:
1. Call CountConversation
2. Call Greater
3. Call Greater

---

### TestTokenProvider_ExpiredToken

**ID**: `UT-LLM-TestTokenProvider_ExpiredToken`

**Package**: `llm`

**File**: `internal/llm/vertexai_provider_test.go:797`

**Description**: No description provided

**Test Steps**:
1. Call Add
2. Call Now
3. Call False
4. Call Valid

---

### TestTokenProvider_GetToken

**ID**: `UT-LLM-TestTokenProvider_GetToken`

**Package**: `llm`

**File**: `internal/llm/vertexai_provider_test.go:775`

**Description**: No description provided

**Test Steps**:
1. Call Add
2. Call Now
3. Call GetToken
4. Call Background
5. Call NoError
   ... and 5 more steps

---

### TestTokenizeQuery

**ID**: `UT-REP-TestTokenizeQuery`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:631`

**Description**: No description provided

**Test Steps**:
1. Call tokenizeQuery
2. Call Errorf

---

### TestTokenizeSymbolName

**ID**: `UT-REP-TestTokenizeSymbolName`

**Package**: `repomap`

**File**: `internal/repomap/repomap_test.go:653`

**Description**: No description provided

**Test Steps**:
1. Call tokenizeSymbolName
2. Call Errorf

---

### TestToolRegistry

**ID**: `UT-TOO-TestToolRegistry`

**Package**: `tools`

**File**: `internal/tools/registry_test.go:11`

**Description**: No description provided

**Test Steps**:
1. Call TempDir
2. Call Fatalf
3. Call Close
4. Call Run
5. Call List
   ... and 5 more steps

---

### TestTrackRequest_CostCalculation

**ID**: `UT-LLM-TestTrackRequest_CostCalculation`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:545`

**Description**: No description provided

**Test Steps**:
1. Call TrackRequest
2. Call GetSessionUsage
3. Call NoError
4. Call InDelta

---

### TestTrackRequest_MultipleRequests

**ID**: `UT-LLM-TestTrackRequest_MultipleRequests`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:462`

**Description**: No description provided

**Test Steps**:
1. Call TrackRequest
2. Call TrackRequest
3. Call TrackRequest
4. Call GetSessionUsage
5. Call NoError
   ... and 5 more steps

---

### TestTrackRequest_MultipleSessions

**ID**: `UT-LLM-TestTrackRequest_MultipleSessions`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:491`

**Description**: No description provided

**Test Steps**:
1. Call TrackRequest
2. Call TrackRequest
3. Call TrackRequest
4. Call GetSessionUsage
5. Call NoError
   ... and 5 more steps

---

### TestTrackRequest_SingleRequest

**ID**: `UT-LLM-TestTrackRequest_SingleRequest`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:441`

**Description**: No description provided

**Test Steps**:
1. Call TrackRequest
2. Call GetSessionUsage
3. Call NoError
4. Call Equal
5. Call Equal
   ... and 5 more steps

---

### TestTrackRequest_ThinkingTokens

**ID**: `UT-LLM-TestTrackRequest_ThinkingTokens`

**Package**: `llm`

**File**: `internal/llm/token_budget_test.go:524`

**Description**: No description provided

**Test Steps**:
1. Call TrackRequest
2. Call GetSessionUsage
3. Call NoError
4. Call Equal

---

### TestTransactionManager_Lifecycle

**ID**: `UT-MUL-TestTransactionManager_Lifecycle`

**Package**: `multiedit`

**File**: `internal/tools/multiedit/multiedit_test.go:15`

**Description**: Test 1: Transaction Lifecycle

**Test Steps**:
1. Call Begin
2. Call Background
3. Call NoError
4. Call Equal
5. Call NotEmpty
   ... and 5 more steps

---

### TestTransactionManager_ListAndCleanup

**ID**: `UT-MUL-TestTransactionManager_ListAndCleanup`

**Package**: `multiedit`

**File**: `internal/tools/multiedit/multiedit_test.go:490`

**Description**: Test 16: Transaction List and Cleanup

**Test Steps**:
1. Call Begin
2. Call Background
3. Call Begin
4. Call Background
5. Call Begin
   ... and 5 more steps

---

### TestTransactionManager_StateTransitions

**ID**: `UT-MUL-TestTransactionManager_StateTransitions`

**Package**: `multiedit`

**File**: `internal/tools/multiedit/multiedit_test.go:140`

**Description**: Test 6: State Transitions

**Test Steps**:
1. Call Begin
2. Call Background
3. Call NoError
4. Call UpdateState
5. Call NoError
   ... and 5 more steps

---

### TestTransactionManager_Timeout

**ID**: `UT-MUL-TestTransactionManager_Timeout`

**Package**: `multiedit`

**File**: `internal/tools/multiedit/multiedit_test.go:330`

**Description**: Test 11: Transaction Timeout

**Test Steps**:
1. Call Begin
2. Call Background
3. Call NoError
4. Call Sleep
5. Call Equal

---

### TestTruncateToTokenBudget_ExceedsBudget

**ID**: `UT-LLM-TestTruncateToTokenBudget_ExceedsBudget`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:627`

**Description**: No description provided

**Test Steps**:
1. Call Repeat
2. Call NotEqual
3. Call Contains
4. Call Less

---

### TestTruncateToTokenBudget_WithinBudget

**ID**: `UT-LLM-TestTruncateToTokenBudget_WithinBudget`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:617`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call NotContains

---

### TestUpdateProjectMetadata

**ID**: `UT-PRO-TestUpdateProjectMetadata`

**Package**: `project`

**File**: `internal/project/manager_test.go:135`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call NoError
3. Call RemoveAll
4. Call CreateProject
5. Call Background
   ... and 5 more steps

---

### TestValidateConfig

**ID**: `UT-CON-TestValidateConfig`

**Package**: `config`

**File**: `internal/config/config_test.go:59`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Error
3. Call NoError

---

### TestValidateReasoningEffort_InvalidLevels

**ID**: `UT-LLM-TestValidateReasoningEffort_InvalidLevels`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:260`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call Error
3. Call Contains
4. Call Error

---

### TestValidateReasoningEffort_ValidLevels

**ID**: `UT-LLM-TestValidateReasoningEffort_ValidLevels`

**Package**: `llm`

**File**: `internal/llm/reasoning_test.go:234`

**Description**: Test ValidateReasoningEffort

**Test Steps**:
1. Call Run
2. Call NoError
3. Call Equal

---

### TestValidateRestore

**ID**: `UT-SNA-TestValidateRestore`

**Package**: `snapshots`

**File**: `internal/workflow/snapshots/snapshots_test.go:666`

**Description**: TestValidateRestore tests restore validation

**Test Steps**:
1. Call Fatalf
2. Call Background
3. Call CreateSnapshot
4. Call Fatalf
5. Call ValidateRestore
   ... and 5 more steps

---

### TestVertexAIProvider_Close

**ID**: `UT-LLM-TestVertexAIProvider_Close`

**Package**: `llm`

**File**: `internal/llm/vertexai_provider_test.go:724`

**Description**: No description provided

**Test Steps**:
1. Call Close
2. Call NoError

---

### TestVertexAIProvider_ErrorHandling

**ID**: `UT-LLM-TestVertexAIProvider_ErrorHandling`

**Package**: `llm`

**File**: `internal/llm/vertexai_provider_test.go:518`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call NewServer
3. Call HandlerFunc
4. Call WriteHeader
5. Call Write
   ... and 5 more steps

---

### TestVertexAIProvider_GenerateClaude

**ID**: `UT-LLM-TestVertexAIProvider_GenerateClaude`

**Package**: `llm`

**File**: `internal/llm/vertexai_provider_test.go:282`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Contains
4. Call Contains
5. Call NotEmpty
   ... and 5 more steps

---

### TestVertexAIProvider_GenerateGemini

**ID**: `UT-LLM-TestVertexAIProvider_GenerateGemini`

**Package**: `llm`

**File**: `internal/llm/vertexai_provider_test.go:216`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Contains
4. Call Contains
5. Call NotEmpty
   ... and 5 more steps

---

### TestVertexAIProvider_GenerateWithSystemInstruction

**ID**: `UT-LLM-TestVertexAIProvider_GenerateWithSystemInstruction`

**Package**: `llm`

**File**: `internal/llm/vertexai_provider_test.go:339`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Decode
4. Call NewDecoder
5. Call NoError
   ... and 5 more steps

---

### TestVertexAIProvider_GenerateWithTools

**ID**: `UT-LLM-TestVertexAIProvider_GenerateWithTools`

**Package**: `llm`

**File**: `internal/llm/vertexai_provider_test.go:391`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Decode
4. Call NewDecoder
5. Call NoError
   ... and 5 more steps

---

### TestVertexAIProvider_GetCapabilities

**ID**: `UT-LLM-TestVertexAIProvider_GetCapabilities`

**Package**: `llm`

**File**: `internal/llm/vertexai_provider_test.go:190`

**Description**: No description provided

**Test Steps**:
1. Call GetCapabilities
2. Call NotEmpty
3. Call True
4. Call True
5. Call True
   ... and 2 more steps

---

### TestVertexAIProvider_GetHealth

**ID**: `UT-LLM-TestVertexAIProvider_GetHealth`

**Package**: `llm`

**File**: `internal/llm/vertexai_provider_test.go:691`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Encode
4. Call NewEncoder
5. Call Close
   ... and 5 more steps

---

### TestVertexAIProvider_GetModels

**ID**: `UT-LLM-TestVertexAIProvider_GetModels`

**Package**: `llm`

**File**: `internal/llm/vertexai_provider_test.go:157`

**Description**: No description provided

**Test Steps**:
1. Call GetModels
2. Call NotEmpty
3. Call Equal
4. Call Greater
5. Call NotEmpty
   ... and 5 more steps

---

### TestVertexAIProvider_GetName

**ID**: `UT-LLM-TestVertexAIProvider_GetName`

**Package**: `llm`

**File**: `internal/llm/vertexai_provider_test.go:152`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call GetName

---

### TestVertexAIProvider_GetType

**ID**: `UT-LLM-TestVertexAIProvider_GetType`

**Package**: `llm`

**File**: `internal/llm/vertexai_provider_test.go:147`

**Description**: No description provided

**Test Steps**:
1. Call Equal
2. Call GetType

---

### TestVertexAIProvider_IsAvailable

**ID**: `UT-LLM-TestVertexAIProvider_IsAvailable`

**Package**: `llm`

**File**: `internal/llm/vertexai_provider_test.go:210`

**Description**: No description provided

**Test Steps**:
1. Call IsAvailable
2. Call Background
3. Call True

---

### TestVertexAIProvider_IsClaudeModel

**ID**: `UT-LLM-TestVertexAIProvider_IsClaudeModel`

**Package**: `llm`

**File**: `internal/llm/vertexai_provider_test.go:753`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call isClaudeModel
3. Call Equal

---

### TestVertexAIProvider_MessageConversion

**ID**: `UT-LLM-TestVertexAIProvider_MessageConversion`

**Package**: `llm`

**File**: `internal/llm/vertexai_provider_test.go:730`

**Description**: No description provided

**Test Steps**:
1. Call convertMessages
2. Call Equal
3. Call Len
4. Call Equal
5. Call Equal
   ... and 1 more steps

---

### TestVertexAIProvider_SafetySettings

**ID**: `UT-LLM-TestVertexAIProvider_SafetySettings`

**Package**: `llm`

**File**: `internal/llm/vertexai_provider_test.go:474`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Decode
4. Call NewDecoder
5. Call NoError
   ... and 5 more steps

---

### TestVertexAIProvider_StreamingClaudeNotSupported

**ID**: `UT-LLM-TestVertexAIProvider_StreamingClaudeNotSupported`

**Package**: `llm`

**File**: `internal/llm/vertexai_provider_test.go:674`

**Description**: No description provided

**Test Steps**:
1. Call New
2. Call GenerateStream
3. Call Background
4. Call Error
5. Call Contains
   ... and 1 more steps

---

### TestVertexAIProvider_StreamingGemini

**ID**: `UT-LLM-TestVertexAIProvider_StreamingGemini`

**Package**: `llm`

**File**: `internal/llm/vertexai_provider_test.go:622`

**Description**: No description provided

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Contains
4. Call Equal
5. Call Get
   ... and 5 more steps

---

### TestVisibility

**ID**: `UT-MAP-TestVisibility`

**Package**: `mapping`

**File**: `internal/tools/mapping/mapping_test.go:188`

**Description**: TestVisibility tests the Visibility enum

**Test Steps**:
1. Call Run
2. Call Equal
3. Call String

---

### TestVoiceError

**ID**: `UT-VOI-TestVoiceError`

**Package**: `voice`

**File**: `internal/tools/voice/voice_test.go:437`

**Description**: TestVoiceError tests error wrapping

**Test Steps**:
1. Call Fatal
2. Call Error
3. Call Error
4. Call IsExist
5. Call Unwrap
   ... and 1 more steps

---

### TestVoiceInputManager_Integration

**ID**: `UT-VOI-TestVoiceInputManager_Integration`

**Package**: `voice`

**File**: `internal/tools/voice/voice_test.go:365`

**Description**: TestVoiceInputManager_Integration tests the full integration

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Fatalf
5. Call Background
   ... and 5 more steps

---

### TestWebTools_FetchAndParse_Integration

**ID**: `UT-WEB-TestWebTools_FetchAndParse_Integration`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:324`

**Description**: Test 15: Fetch and parse integration

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Set
4. Call Header
5. Call WriteHeader
   ... and 5 more steps

---

### TestWebTools_Fetch_EmptyURL

**ID**: `UT-WEB-TestWebTools_Fetch_EmptyURL`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:539`

**Description**: Test 23: Empty URL error

**Test Steps**:
1. Call NoError
2. Call Close
3. Call Fetch
4. Call Background
5. Call Error
   ... and 2 more steps

---

### TestWebTools_Fetch_WithCache

**ID**: `UT-WEB-TestWebTools_Fetch_WithCache`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:360`

**Description**: Test 16: Caching with fetch

**Test Steps**:
1. Call NewServer
2. Call HandlerFunc
3. Call Set
4. Call Header
5. Call WriteHeader
   ... and 5 more steps

---

### TestWebTools_Search_EmptyQuery

**ID**: `UT-WEB-TestWebTools_Search_EmptyQuery`

**Package**: `web`

**File**: `internal/tools/web/web_test.go:527`

**Description**: Test 22: Empty search query error

**Test Steps**:
1. Call NoError
2. Call Close
3. Call Search
4. Call Background
5. Call Error
   ... and 2 more steps

---

### TestWholeEditorApply

**ID**: `UT-EDI-TestWholeEditorApply`

**Package**: `editor`

**File**: `internal/editor/whole_editor_test.go:10`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Repeat
5. Call Run
   ... and 5 more steps

---

### TestWholeEditorGetFileStats

**ID**: `UT-EDI-TestWholeEditorGetFileStats`

**Package**: `editor`

**File**: `internal/editor/whole_editor_test.go:392`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call GetFileStats
3. Call Errorf
4. Call Errorf
5. Call Errorf
   ... and 1 more steps

---

### TestWholeEditorInvalidContent

**ID**: `UT-EDI-TestWholeEditorInvalidContent`

**Package**: `editor`

**File**: `internal/editor/whole_editor_test.go:461`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Join
5. Call Apply
   ... and 1 more steps

---

### TestWholeEditorNewFile

**ID**: `UT-EDI-TestWholeEditorNewFile`

**Package**: `editor`

**File**: `internal/editor/whole_editor_test.go:89`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Join
5. Call Apply
   ... and 4 more steps

---

### TestWholeEditorValidateGoSyntax

**ID**: `UT-EDI-TestWholeEditorValidateGoSyntax`

**Package**: `editor`

**File**: `internal/editor/whole_editor_test.go:122`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call validateGoSyntax
3. Call Error
4. Call Errorf

---

### TestWholeEditorValidateJSONSyntax

**ID**: `UT-EDI-TestWholeEditorValidateJSONSyntax`

**Package**: `editor`

**File**: `internal/editor/whole_editor_test.go:194`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call validateJSONSyntax
3. Call Error
4. Call Errorf

---

### TestWholeEditorValidateSyntax

**ID**: `UT-EDI-TestWholeEditorValidateSyntax`

**Package**: `editor`

**File**: `internal/editor/whole_editor_test.go:327`

**Description**: No description provided

**Test Steps**:
1. Call MkdirTemp
2. Call Fatalf
3. Call RemoveAll
4. Call Run
5. Call Join
   ... and 3 more steps

---

### TestWholeEditorValidateYAMLSyntax

**ID**: `UT-EDI-TestWholeEditorValidateYAMLSyntax`

**Package**: `editor`

**File**: `internal/editor/whole_editor_test.go:260`

**Description**: No description provided

**Test Steps**:
1. Call Run
2. Call validateYAMLSyntax
3. Call Error
4. Call Errorf

---

### TestWorkerCapabilities

**ID**: `UT-WOR-TestWorkerCapabilities`

**Package**: `worker`

**File**: `internal/worker/distributed_manager_test.go:155`

**Description**: TestWorkerCapabilities tests worker capability matching

**Test Steps**:
1. Call Background
2. Call Initialize
3. Call Log
4. Call GetAvailableWorkers
5. Call Logf
   ... and 4 more steps

---

### TestWorkerConfigValidation

**ID**: `UT-WOR-TestWorkerConfigValidation`

**Package**: `worker`

**File**: `internal/worker/distributed_manager_test.go:52`

**Description**: TestWorkerConfigValidation tests worker configuration validation

**Test Steps**:
1. Call Run
2. Call Background
3. Call Initialize
4. Call Error
5. Call Errorf

---

### TestWorkerHealthMonitoring

**ID**: `UT-WOR-TestWorkerHealthMonitoring`

**Package**: `worker`

**File**: `internal/worker/distributed_manager_test.go:248`

**Description**: TestWorkerHealthMonitoring tests worker health monitoring

**Test Steps**:
1. Call GetAvailableWorkers
2. Call Skip
3. Call Error
4. Call IsZero
5. Call Error
   ... and 2 more steps

---

### TestWorkflowAddStep

**ID**: `UT-AGE-TestWorkflowAddStep`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:29`

**Description**: No description provided

**Test Steps**:
1. Call AddStep
2. Call AddStep
3. Call Len
4. Call Equal
5. Call Equal

---

### TestWorkflowComplexDependencies

**ID**: `UT-AGE-TestWorkflowComplexDependencies`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:685`

**Description**: TestWorkflowComplexDependencies tests workflows with complex dependency chains

**Test Steps**:
1. Call AddStep
2. Call AddStep
3. Call AddStep
4. Call AddStep
5. Call GetReadySteps
   ... and 5 more steps

---

### TestWorkflowConcurrentStateModification

**ID**: `UT-AGE-TestWorkflowConcurrentStateModification`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:932`

**Description**: TestWorkflowConcurrentStateModification tests thread-safety

**Test Steps**:
1. Call Sprintf
2. Call Sprintf
3. Call AddStep
4. Call Len
5. Call SetStepResult
   ... and 3 more steps

---

### TestWorkflowDuplicateStepID

**ID**: `UT-AGE-TestWorkflowDuplicateStepID`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:857`

**Description**: TestWorkflowDuplicateStepID tests handling of duplicate step IDs

**Test Steps**:
1. Call AddStep
2. Call AddStep
3. Call Len
4. Call SetStepResult
5. Call GetReadySteps
   ... and 1 more steps

---

### TestWorkflowEmptyWorkflow

**ID**: `UT-AGE-TestWorkflowEmptyWorkflow`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:838`

**Description**: TestWorkflowEmptyWorkflow tests empty workflow edge case

**Test Steps**:
1. Call Len
2. Call GetReadySteps
3. Call Len
4. Call Start
5. Call Equal
   ... and 2 more steps

---

### TestWorkflowExecutorCapabilityMatching

**ID**: `UT-AGE-TestWorkflowExecutorCapabilityMatching`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:648`

**Description**: No description provided

**Test Steps**:
1. Call RegisterAgent
2. Call AddStep
3. Call Background
4. Call ExecuteWorkflow
5. Call NoError
   ... and 4 more steps

---

### TestWorkflowExecutorContextCancellation

**ID**: `UT-AGE-TestWorkflowExecutorContextCancellation`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:503`

**Description**: No description provided

**Test Steps**:
1. Call Sleep
2. Call NewResult
3. Call RegisterAgent
4. Call AddStep
5. Call WithTimeout
   ... and 5 more steps

---

### TestWorkflowExecutorGetWorkflow

**ID**: `UT-AGE-TestWorkflowExecutorGetWorkflow`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:605`

**Description**: No description provided

**Test Steps**:
1. Call Lock
2. Call Unlock
3. Call GetWorkflow
4. Call NoError
5. Call Equal
   ... and 2 more steps

---

### TestWorkflowExecutorInputChaining

**ID**: `UT-AGE-TestWorkflowExecutorInputChaining`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:538`

**Description**: No description provided

**Test Steps**:
1. Call NewResult
2. Call SetSuccess
3. Call NewResult
4. Call SetSuccess
5. Call RegisterAgent
   ... and 5 more steps

---

### TestWorkflowExecutorListWorkflows

**ID**: `UT-AGE-TestWorkflowExecutorListWorkflows`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:626`

**Description**: No description provided

**Test Steps**:
1. Call Lock
2. Call Unlock
3. Call ListWorkflows
4. Call Len
5. Call Contains
   ... and 1 more steps

---

### TestWorkflowExecutorMissingAgent

**ID**: `UT-AGE-TestWorkflowExecutorMissingAgent`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:480`

**Description**: No description provided

**Test Steps**:
1. Call AddStep
2. Call Background
3. Call ExecuteWorkflow
4. Call Error
5. Call Equal

---

### TestWorkflowExecutorOptionalStep

**ID**: `UT-AGE-TestWorkflowExecutorOptionalStep`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:407`

**Description**: No description provided

**Test Steps**:
1. Call Errorf
2. Call RegisterAgent
3. Call RegisterAgent
4. Call RegisterAgent
5. Call AddStep
   ... and 5 more steps

---

### TestWorkflowExecutorParallelSteps

**ID**: `UT-AGE-TestWorkflowExecutorParallelSteps`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:349`

**Description**: No description provided

**Test Steps**:
1. Call RegisterAgent
2. Call RegisterAgent
3. Call RegisterAgent
4. Call AddStep
5. Call AddStep
   ... and 5 more steps

---

### TestWorkflowExecutorSimpleWorkflow

**ID**: `UT-AGE-TestWorkflowExecutorSimpleWorkflow`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:291`

**Description**: No description provided

**Test Steps**:
1. Call RegisterAgent
2. Call NoError
3. Call RegisterAgent
4. Call NoError
5. Call AddStep
   ... and 5 more steps

---

### TestWorkflowGetReadySteps

**ID**: `UT-AGE-TestWorkflowGetReadySteps`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:194`

**Description**: No description provided

**Test Steps**:
1. Call AddStep
2. Call AddStep
3. Call AddStep
4. Call AddStep
5. Call GetReadySteps
   ... and 5 more steps

---

### TestWorkflowIsStepReady

**ID**: `UT-AGE-TestWorkflowIsStepReady`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:103`

**Description**: No description provided

**Test Steps**:
1. Call AddStep
2. Call AddStep
3. Call AddStep
4. Call True
5. Call IsStepReady
   ... and 5 more steps

---

### TestWorkflowIsStepReadyWithOptionalDependency

**ID**: `UT-AGE-TestWorkflowIsStepReadyWithOptionalDependency`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:163`

**Description**: No description provided

**Test Steps**:
1. Call AddStep
2. Call AddStep
3. Call SetStepResult
4. Call True
5. Call IsStepReady

---

### TestWorkflowLargeScale

**ID**: `UT-AGE-TestWorkflowLargeScale`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:765`

**Description**: TestWorkflowLargeScale tests workflow with many steps

**Test Steps**:
1. Call Sprintf
2. Call Sprintf
3. Call AddStep
4. Call GetReadySteps
5. Call Len
   ... and 5 more steps

---

### TestWorkflowLinearChain

**ID**: `UT-AGE-TestWorkflowLinearChain`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:796`

**Description**: TestWorkflowLinearChain tests a long linear dependency chain

**Test Steps**:
1. Call Sprintf
2. Call Sprintf
3. Call Sprintf
4. Call AddStep
5. Call GetReadySteps
   ... and 5 more steps

---

### TestWorkflowMultipleOptionalSteps

**ID**: `UT-AGE-TestWorkflowMultipleOptionalSteps`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:745`

**Description**: TestWorkflowMultipleOptionalSteps tests multiple optional steps

**Test Steps**:
1. Call AddStep
2. Call AddStep
3. Call AddStep
4. Call SetStepResult
5. Call SetStepResult
   ... and 2 more steps

---

### TestWorkflowNilDependencies

**ID**: `UT-AGE-TestWorkflowNilDependencies`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:878`

**Description**: TestWorkflowNilDependencies tests steps with nil dependency slices

**Test Steps**:
1. Call AddStep
2. Call True
3. Call IsStepReady
4. Call GetReadySteps
5. Call Len

---

### TestWorkflowRequiredStepFailure

**ID**: `UT-AGE-TestWorkflowRequiredStepFailure`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:724`

**Description**: TestWorkflowRequiredStepFailure tests that workflow fails when required step fails

**Test Steps**:
1. Call AddStep
2. Call AddStep
3. Call SetStepResult
4. Call False
5. Call IsStepReady
   ... and 2 more steps

---

### TestWorkflowSetGetStepResult

**ID**: `UT-AGE-TestWorkflowSetGetStepResult`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:83`

**Description**: No description provided

**Test Steps**:
1. Call Now
2. Call SetStepResult
3. Call GetStepResult
4. Call True
5. Call Equal
   ... and 2 more steps

---

### TestWorkflowStateTransitions

**ID**: `UT-AGE-TestWorkflowStateTransitions`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:54`

**Description**: No description provided

**Test Steps**:
1. Call Start
2. Call Equal
3. Call NotNil
4. Call Nil
5. Call Complete
   ... and 5 more steps

---

### TestWorkflowStepInputMerging

**ID**: `UT-AGE-TestWorkflowStepInputMerging`

**Package**: `agent`

**File**: `internal/agent/workflow_test.go:898`

**Description**: TestWorkflowStepInputMerging tests input merging from dependencies

**Test Steps**:
1. Call AddStep
2. Call AddStep
3. Call SetStepResult
4. Call True
5. Call IsStepReady

---

### TestYandexMessengerChannel

**ID**: `UT-NOT-TestYandexMessengerChannel`

**Package**: `notification`

**File**: `internal/notification/engine_test.go:116`

**Description**: No description provided

**Test Steps**:
1. Call NotNil
2. Call Equal
3. Call GetName
4. Call True
5. Call IsEnabled
   ... and 3 more steps

---

## Usage

To run a specific test:

```bash
# Run by package
go test -v ./internal/package_name

# Run specific test
go test -v ./path/to/package -run TestName

# Run all tests
./run_all_tests.sh
```

## Automatic Updates

This catalog is automatically generated by running:

```bash
go run scripts/generate-test-catalog.go
```

The catalog should be regenerated whenever new tests are added.

