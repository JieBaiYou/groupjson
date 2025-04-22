#!/bin/bash

# 设置颜色输出
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}       GroupJSON 使用示例演示              ${NC}"
echo -e "${BLUE}============================================${NC}"

# 步骤1: 显示当前目录结构
echo -e "\n${GREEN}步骤1: 查看源代码文件${NC}"
echo "当前目录包含以下Go源文件:"
ls -la *.go | grep -v "_groupjson.go"

# 步骤2: 注释说明
echo -e "\n${GREEN}步骤2: 关于实现${NC}"
echo "这个示例使用了手动添加的方法来模拟代码生成的效果,"
echo "当GroupJSON的代码生成器模板问题修复后, 可以使用代码生成来获得更高性能。"
echo "目前示例中已经在源文件中包含了被注释的//go:generate指令供参考。"

# 步骤3: 编译并运行示例
echo -e "\n${GREEN}步骤3: 运行示例程序${NC}"
echo "执行: go run ."
go run .

echo -e "\n${BLUE}============================================${NC}"
echo -e "${BLUE}       演示完成!                           ${NC}"
echo -e "${BLUE}============================================${NC}"
echo -e "\n提示: 这个示例展示了如何使用GroupJSON按组选择性地序列化结构体字段"
echo -e "您可以查看README.md获取更多信息"