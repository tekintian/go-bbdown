# Go-BBDown 单元测试

本目录包含了 go-bbdown 项目的所有单元测试。

## 目录结构

```
tests/
├── README.md          # 本文档
├── util/              # util 包的测试
│   └── string_test.go # 字符串处理工具函数的测试
└── ...                # 其他包的测试（将来添加）
```

## 运行测试

### 运行所有测试

使用项目根目录的测试脚本：

```bash
./run_tests.sh
```

### 运行特定包的测试

```bash
# 运行 util 包的测试
go test ./tests/util -v

# 运行测试并显示覆盖率
go test ./tests/util -cover
```

### 运行特定测试函数

```bash
# 只运行 ExtractFromURL 相关的测试
go test ./tests/util -v -run TestExtractFromURL

# 只运行 BV 转换相关的测试
go test ./tests/util -v -run TestBVConverter
```

## 测试覆盖的功能

### util/string_test.go

目前测试了以下功能：

1. **URL解析功能** (`TestExtractFromURL`)
   - BV号URL
   - AV号URL
   - EP号URL
   - SS号URL
   - 合集URL（带参数和不带参数）
   - 收藏夹URL
   - 媒体列表URL
   - 课程URL
   - 直接ID输入

2. **视频ID提取功能** (`TestExtractVideoID`)
   - 从各种URL格式中提取标准化的视频ID
   - 处理直接输入的ID

3. **BV转换器功能** (`TestBVConverter`)
   - AV号转BV号
   - 错误处理

4. **ID验证功能** (`TestIsValidID`)
   - 验证各种ID格式的有效性
   - 边界情况测试

5. **查询字符串解析** (`TestGetQueryString`)
   - 从URL中提取查询参数
   - 各种特殊情况处理

## 编写新的测试

1. 在对应的包目录下创建 `*_test.go` 文件
2. 文件名以 `_test.go` 结尾
3. 测试函数以 `Test` 开头，接受 `*testing.T` 参数
4. 使用标准Go测试框架编写测试用例

### 示例

```go
package your_package_test

import (
    "testing"
    "github.com/tekintian/go-bbdown/your_package"
)

func TestYourFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "测试用例1",
            input:    "test input",
            expected: "expected output",
            wantErr:  false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := your_package.YourFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("YourFunction() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.expected {
                t.Errorf("YourFunction() = %v, want %v", got, tt.expected)
            }
        })
    }
}
```

## 测试最佳实践

1. **命名规范**：测试函数名应该清楚地描述测试的内容
2. **表格驱动测试**：使用结构体切片来组织测试用例，便于添加新的测试场景
3. **错误处理**：测试应该覆盖正常情况和错误情况
4. **边界条件**：测试边界值和特殊情况
5. **独立性**：每个测试应该是独立的，不依赖其他测试的状态

## 持续集成

测试可以在CI/CD流水线中自动运行，确保代码质量：

```bash
# 在CI中运行所有测试
go test ./tests/... -v

# 生成测试覆盖率报告
go test ./tests/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```