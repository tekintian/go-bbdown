#!/bin/bash

# 运行所有单元测试的脚本

echo "运行 go-bbdown 单元测试..."
echo "================================"

# 运行util包的测试
echo "运行 util 包测试..."
go test ./tests/util -v

# 检查测试结果
if [ $? -eq 0 ]; then
    echo "================================"
    echo "所有测试通过！"
else
    echo "================================"
    echo "测试失败！"
    exit 1
fi

# 运行项目中的其他测试（如果有的话）
echo ""
echo "检查项目中是否有其他测试..."
if [ -d "core" ]; then
    echo "运行 core 包测试..."
    go test ./core -v
fi

if [ -d "cmd" ]; then
    echo "运行 cmd 包测试..."
    go test ./cmd -v
fi

echo ""
echo "运行覆盖率统计..."
go test ./tests/util -cover

echo ""
echo "测试完成！"