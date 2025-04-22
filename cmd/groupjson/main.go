package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/JieBaiYou/groupjson"
)

func main() {
	// 定义命令行参数
	var (
		typeName   = flag.String("type", "", "目标结构体名称, 必须指定")
		sourceFile = flag.String("source", "", "源文件路径, 默认与type同名")
		outputFile = flag.String("output", "", "输出文件路径, 默认为<type>_groupjson.go")
		tagName    = flag.String("tag", "groups", "分组标签名, 默认为groups")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: groupjson -type=TypeName [options]\n\n")
		fmt.Fprintf(os.Stderr, "选项:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// 验证必要参数
	if *typeName == "" {
		flag.Usage()
		os.Exit(1)
	}

	// 如果未指定源文件, 使用与类型相同的名称
	if *sourceFile == "" {
		*sourceFile = strings.ToLower(*typeName) + ".go"
	}

	// 初始化生成器
	gen := groupjson.NewGenerator()
	gen.TypeName = *typeName
	gen.SourceFile = *sourceFile
	gen.OutputFile = *outputFile
	gen.TagName = *tagName

	// 运行代码生成
	err := gen.Generate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated code for %s at %s\n", *typeName, gen.OutputFile)
}
