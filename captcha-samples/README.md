# 微博图形验证码 OpenCV 实验

采集了 4 组网易易盾图标点选验证码。`01.jpg` 至 `03.jpg` 用于本地调试，`04.jpg` 用于修改算法后的页面验证。

## 运行

使用 Go 命令行识别：

```sh
go run ./cmd/yidun-iconclick captcha-samples/01.jpg captcha-samples/02.jpg
go run ./cmd/yidun-iconclick captcha-samples/04.jpg
```

标准输出为 JSON，包含坐标、相关性分数、旋转角度及缩放比例。相关性分数不是正确概率。目录中已有的 `*-matches.png`、`results.json` 和 `validation-results.json` 是原实验保留的标注及结果，Go 命令不会重新生成这些历史文件。

## 方法与适用范围

这些原始图片为 480×360：上方 480×240 是场景，底部是提示图标素材。脚本从底部第一行按顺序截取三个提示图标，灰度阈值提取形状；对场景和模板做高斯背景相减，再枚举旋转和缩放，使用 OpenCV 归一化相关性匹配，同时考虑深色和浅色目标。

目前尺寸、提示图标数量和素材排列均针对这些样本，尚未泛化到其他布局。前三组最终定位的 9 个目标经目视核对一致；这不是独立测试准确率。

## 页面验证

首次提交因纵坐标换算错误未通过：原图包含底部素材区，不能把整个图片元素高度视为场景高度。

修正后使用第 4 组 OpenCV 输出坐标，按以下公式映射到浏览器 CSS 坐标：

```text
scale = image.getBoundingClientRect().width / image.naturalWidth
clientX = imageRect.x + sourceX * scale
clientY = imageRect.y + sourceY * scale
```

第 4 组原图坐标依次为 `(317.5, 130.5)`、`(129.5, 147.5)`、`(143.5, 57.5)`。

提交后验证框关闭，微博页面显示“57s后重新发送”，并发起短信发送请求。本次在线验证通过，未继续短信登录。单次通过不足以证明长期稳定性。
