// yidun-iconclick 从本地原始图片识别网易易盾图标并输出 JSON。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/leetpy/captchaset/recognizers/yidun/iconclick"
)

// report 关联输入文件与按提示顺序排列的识别结果。
type report struct {
	File    string            `json:"file"`
	Matches []iconclick.Match `json:"matches"`
}

// main 解析命令行并执行本地识别，失败时向标准错误输出原因并以非零状态退出。
func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "错误：", err)
		os.Exit(1)
	}
}

// run 解析 args 中的文件路径，向 stdout 写入 JSON、向 stderr 写入用法，返回读取或识别错误。
func run(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("yidun-iconclick", flag.ContinueOnError)
	flags.SetOutput(stderr)
	// Usage 向 stderr 打印当前识别模块的调用格式，无参数或返回值。
	flags.Usage = func() {
		fmt.Fprintln(stderr, "用法：yidun-iconclick 图片.jpg [图片2.jpg ...]")
	}
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if flags.NArg() == 0 {
		flags.Usage()
		return fmt.Errorf("请指定至少一张 480×360 原始图片")
	}
	reports := make([]report, 0, flags.NArg())
	for _, path := range flags.Args() {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("读取 %q: %w", path, err)
		}
		matches, err := iconclick.Recognize(data)
		if err != nil {
			return fmt.Errorf("识别 %q: %w", path, err)
		}
		reports = append(reports, report{File: path, Matches: matches})
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(reports)
}
