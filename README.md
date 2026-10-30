# captchaset

按厂商和验证码类型拆分的 Go 图片识别模块集合。当前模块：

| 模块 | 用途 | 输入 |
| --- | --- | --- |
| `recognizers/yidun/iconclick` | 网易易盾图标顺序点选，在微博登录中采集验证 | 480×360 原始图片，含底部提示素材 |

模块内包含算法、结果类型和测试；对应命令行位于 `cmd/yidun-iconclick`。新增验证码类型时建立独立目录，不必修改当前模块。当前没有网络请求、浏览器操作或短信发送逻辑。

## 依赖

Go 1.22+、CGO、C/C++ 工具链、OpenCV 4、pkg-config。GoCV 固定为 v0.43.0，本机环境为 OpenCV 4.13.0。

macOS 可使用：

```sh
brew install opencv pkg-config
go mod download
```

OpenCV 为本机动态库依赖，编译运行时均需可用。

## 命令行

```sh
go run ./cmd/yidun-iconclick captcha-samples/04.jpg
go run ./cmd/yidun-iconclick captcha-samples/01.jpg captcha-samples/02.jpg > results.json
go build -o bin/yidun-iconclick ./cmd/yidun-iconclick
```

输出 JSON 数组，每张图片包含三个按提示顺序排列的 `matches`。每项包含 `order`、`point: [x, y]`、`score`、`angle`、`scale`、`box: [x, y, width, height]`。坐标基于原始图片，分数是绝对相关系数，不是正确概率。错误写入标准错误并以状态码 1 退出。

## Go 调用

代码推送到 GitHub 后，其他项目可安装并引用：

```sh
go get github.com/leetpy/captchaset/recognizers/yidun/iconclick
```

```go
package main

import (
    "fmt"
    "os"

    "github.com/leetpy/captchaset/recognizers/yidun/iconclick"
)

// main 读取本地原图并打印按提示顺序排列的识别坐标。
func main() {
    data, err := os.ReadFile("captcha-samples/04.jpg")
    if err != nil {
        panic(err)
    }
    matches, err := iconclick.Recognize(data)
    if err != nil {
        panic(err)
    }
    for _, match := range matches {
        fmt.Printf("%d: %v score=%.4f\n", match.Order, match.Point, match.Score)
    }
}
```

## 测试与范围

```sh
go test ./...
```

使用四组已保存的真实样本检查目标顺序和位置，并检查无效输入。布局说明见 [模块文档](recognizers/yidun/iconclick/README.md)。
