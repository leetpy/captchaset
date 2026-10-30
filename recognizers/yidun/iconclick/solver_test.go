package iconclick

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// TestRecognizeSamples 使用人工核对的 expected 目标位置检查四组样本；t 提供测试断言与结果报告。
func TestRecognizeSamples(t *testing.T) {
	cases := []struct {
		file     string
		expected [3][2]float64
	}{
		{file: "01.jpg", expected: [3][2]float64{{157, 156}, {74, 171}, {388, 164}}},
		{file: "02.jpg", expected: [3][2]float64{{123, 53}, {211, 63}, {414, 159}}},
		{file: "03.jpg", expected: [3][2]float64{{284, 107}, {157, 99}, {408, 167}}},
		{file: "04.jpg", expected: [3][2]float64{{318, 131}, {130, 148}, {144, 58}}},
	}
	for _, tc := range cases {
		// 子测试用当前 tc 的人工标注检查目标顺序与坐标；t 是该子测试的报告对象。
		t.Run(tc.file, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "..", "captcha-samples", tc.file))
			if err != nil {
				t.Fatal(err)
			}
			matches, err := Recognize(data)
			if err != nil {
				t.Fatal(err)
			}
			if len(matches) != 3 {
				t.Fatalf("目标数 = %d，希望为 3", len(matches))
			}
			for index, match := range matches {
				if match.Order != index+1 {
					t.Errorf("目标顺序错误：%+v", match)
				}
				for axis := 0; axis < 2; axis++ {
					if math.Abs(match.Point[axis]-tc.expected[index][axis]) >= 8 {
						t.Errorf("目标 %d 坐标 %v 不在人工标注 %v 附近", index+1, match.Point, tc.expected[index])
					}
				}
			}
		})
	}
}

// TestRecognizeInvalid 使用无效数据及不支持的布局检查错误处理；t 提供测试断言。
func TestRecognizeInvalid(t *testing.T) {
	for _, data := range [][]byte{{}, []byte("invalid"), blankPNG(t, 480, 240), blankPNG(t, 480, 360)} {
		if _, err := Recognize(data); err == nil {
			t.Error("无效输入应返回错误")
		}
	}
}

// blankPNG 生成宽 width、高 height 的纯白 PNG；t 用于报告编码错误，返回编码后的图片字节。
func blankPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	im := image.NewGray(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			im.SetGray(x, y, color.Gray{Y: 255})
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, im); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
