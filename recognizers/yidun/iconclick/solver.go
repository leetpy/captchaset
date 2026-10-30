// Package iconclick 识别已采集的网易易盾图标顺序点选验证码，不包含网络或浏览器操作。
package iconclick

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"

	"gocv.io/x/gocv"
)

// Match 表示一个按提示顺序排列的目标；Point 和 Box 使用原始图片像素坐标。
// Score 是绝对归一化相关系数，不是识别正确概率。
type Match struct {
	Order int        `json:"order"`
	Point [2]float64 `json:"point"`
	Score float64    `json:"score"`
	Angle int        `json:"angle"`
	Scale float64    `json:"scale"`
	Box   [4]int     `json:"box"`
}

// Recognize 识别 data 中 JPEG 或 PNG 原图，返回三个按点击顺序排列的目标或错误。
// data 必须编码一张 480×360 图片，顶部 480×240 为场景，底部首行为三个提示图标。
func Recognize(data []byte) ([]Match, error) {
	if len(data) == 0 {
		return nil, errors.New("图片数据为空")
	}
	source, err := gocv.IMDecode(data, gocv.IMReadColor)
	if err != nil {
		return nil, fmt.Errorf("解码图片: %w", err)
	}
	defer source.Close()
	if source.Empty() {
		return nil, errors.New("无法解码图片")
	}
	if source.Cols() != 480 || source.Rows() != 360 {
		return nil, fmt.Errorf("需要 480×360 原图（含底部素材），实际为 %d×%d", source.Cols(), source.Rows())
	}

	region := source.Region(image.Rect(0, 0, 480, 240))
	defer region.Close()
	gray := gocv.NewMat()
	defer gray.Close()
	if err := gocv.CvtColor(region, &gray, gocv.ColorBGRToGray); err != nil {
		return nil, err
	}
	scene, err := highPass(gray)
	if err != nil {
		return nil, err
	}
	defer scene.Close()

	matches := make([]Match, 0, 3)
	for index := 0; index < 3; index++ {
		match, err := recognizeIcon(source, scene, index)
		if err != nil {
			return nil, fmt.Errorf("提示图标 %d: %w", index+1, err)
		}
		matches = append(matches, match)
	}
	return matches, nil
}

// highPass 对单通道 src 转为浮点并减去高斯背景，返回需由调用者关闭的 Mat 或错误。
func highPass(src gocv.Mat) (gocv.Mat, error) {
	result := gocv.NewMat()
	if err := src.ConvertTo(&result, gocv.MatTypeCV32F); err != nil {
		result.Close()
		return gocv.Mat{}, err
	}
	blurred := gocv.NewMat()
	defer blurred.Close()
	err := gocv.GaussianBlur(
		result,
		&blurred,
		image.Point{},
		5,
		5,
		gocv.BorderDefault,
	)
	if err != nil {
		result.Close()
		return gocv.Mat{}, err
	}
	if err := gocv.Subtract(result, blurred, &result); err != nil {
		result.Close()
		return gocv.Mat{}, err
	}
	return result, nil
}

// recognizeIcon 从 source 提取第 index 个提示（从零计数），在去背景的 scene 中搜索并返回最佳匹配。
func recognizeIcon(source, scene gocv.Mat, index int) (Match, error) {
	tile := source.Region(image.Rect(index*42, 240, index*42+30, 270))
	defer tile.Close()
	gray := gocv.NewMat()
	defer gray.Close()
	if err := gocv.CvtColor(tile, &gray, gocv.ColorBGRToGray); err != nil {
		return Match{}, err
	}
	mask := gocv.NewMat()
	defer mask.Close()
	// THRESH_BINARY_INV 包含阈值本身，因此用 109 表示 gray < 110。
	gocv.Threshold(
		gray,
		&mask,
		109,
		255,
		gocv.ThresholdBinaryInv,
	)
	if gocv.CountNonZero(mask) < 10 {
		return Match{}, errors.New("提示为空或布局不支持")
	}
	best := Match{Order: index + 1, Score: -1}
	for step := 13; step <= 17; step++ {
		scale := float64(step) / 10
		canvas, err := scaledCanvas(mask, scale)
		if err != nil {
			return Match{}, err
		}
		err = searchRotations(scene, canvas, scale, &best)
		canvas.Close()
		if err != nil {
			return Match{}, err
		}
	}
	return best, nil
}

// scaledCanvas 将 mask 按 scale 缩放并居中补零，返回需由调用者关闭的旋转画布或错误。
func scaledCanvas(mask gocv.Mat, scale float64) (gocv.Mat, error) {
	size := int(math.RoundToEven(30 * scale))
	scaled := gocv.NewMat()
	defer scaled.Close()
	err := gocv.Resize(
		mask,
		&scaled,
		image.Pt(size, size),
		0,
		0,
		gocv.InterpolationLinear,
	)
	if err != nil {
		return gocv.Mat{}, err
	}
	canvasSize := int(math.RoundToEven(float64(size) * 1.5))
	// 显式清零，避免未初始化的背景污染旋转及模板匹配。
	canvas := gocv.NewMatWithSizeFromScalar(
		gocv.Scalar{},
		canvasSize,
		canvasSize,
		gocv.MatTypeCV8U,
	)
	offset := (canvasSize - size) / 2
	region := canvas.Region(image.Rect(offset, offset, offset+size, offset+size))
	defer region.Close()
	if err := scaled.CopyTo(&region); err != nil {
		canvas.Close()
		return gocv.Mat{}, err
	}
	return canvas, nil
}

// searchRotations 枚举 canvas 的旋转角度，按 scene 的匹配分数更新 best；scale 记录当前缩放比，返回处理错误。
func searchRotations(scene, canvas gocv.Mat, scale float64, best *Match) error {
	for angle := 0; angle < 360; angle += 5 {
		template, err := rotatedTemplate(canvas, angle)
		if err != nil {
			return err
		}
		candidate, err := matchTemplate(scene, template)
		template.Close()
		if err != nil {
			return err
		}
		if candidate.Score > best.Score {
			candidate.Order = best.Order
			candidate.Angle = angle
			candidate.Scale = scale
			*best = candidate
		}
	}
	return nil
}

// rotatedTemplate 将 canvas 旋转 angle 度、裁剪并去背景，返回需由调用者关闭的模板或错误。
func rotatedTemplate(canvas gocv.Mat, angle int) (gocv.Mat, error) {
	rotation := gocv.GetRotationMatrix2D(image.Point{}, float64(angle), 1)
	defer rotation.Close()
	// GoCV 的旋转中心是整数 Point；直接设置平移量以保留半像素中心。
	center := float64(canvas.Cols()) / 2
	alpha := rotation.GetDoubleAt(0, 0)
	beta := rotation.GetDoubleAt(0, 1)
	rotation.SetDoubleAt(0, 2, (1-alpha)*center-beta*center)
	rotation.SetDoubleAt(1, 2, beta*center+(1-alpha)*center)
	rotated := gocv.NewMat()
	defer rotated.Close()
	err := gocv.WarpAffine(
		canvas,
		&rotated,
		rotation,
		image.Pt(canvas.Cols(), canvas.Rows()),
	)
	if err != nil {
		return gocv.Mat{}, err
	}
	points := gocv.NewMat()
	defer points.Close()
	if err := gocv.FindNonZero(rotated, &points); err != nil {
		return gocv.Mat{}, err
	}
	if points.Empty() {
		return gocv.Mat{}, errors.New("旋转后模板为空")
	}
	vector := gocv.NewPointVectorFromMat(points)
	defer vector.Close()
	region := rotated.Region(gocv.BoundingRect(vector))
	defer region.Close()
	padded := gocv.NewMat()
	defer padded.Close()
	err = gocv.CopyMakeBorder(
		region,
		&padded,
		8,
		8,
		8,
		8,
		gocv.BorderConstant,
		color.RGBA{},
	)
	if err != nil {
		return gocv.Mat{}, err
	}
	return highPass(padded)
}

// matchTemplate 使用 template 在 scene 中搜索深色和浅色目标，返回最佳匹配或处理错误。
func matchTemplate(scene, template gocv.Mat) (Match, error) {
	scores := gocv.NewMat()
	defer scores.Close()
	mask := gocv.NewMat()
	defer mask.Close()
	err := gocv.MatchTemplate(
		scene,
		template,
		&scores,
		gocv.TmCcoeffNormed,
		mask,
	)
	if err != nil {
		return Match{}, err
	}
	low, high, lowLocation, highLocation := gocv.MinMaxLoc(scores)
	score, location := float64(high), highLocation
	if -low > high {
		score, location = float64(-low), lowLocation
	}
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return Match{}, errors.New("模板匹配产生无效分数")
	}
	w, h := template.Cols(), template.Rows()
	return Match{
		Score: score,
		Point: [2]float64{float64(location.X) + float64(w)/2, float64(location.Y) + float64(h)/2},
		Box:   [4]int{location.X, location.Y, w, h},
	}, nil
}
