package upscaler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/image/draw"
)

// Upscaler 이미지 업스케일러
type Upscaler struct {
	scale int
}

// NewUpscaler 새 업스케일러 생성
func NewUpscaler(scale int) *Upscaler {
	if scale < 2 {
		scale = 2
	}
	if scale > 4 {
		scale = 4
	}
	return &Upscaler{scale: scale}
}

// UpscaleDirectory 디렉토리 내 모든 이미지 업스케일
func (u *Upscaler) UpscaleDirectory(inputDir, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	files, err := findImageFiles(inputDir)
	if err != nil {
		return err
	}

	fmt.Printf("🔍 %d개 이미지 발견\n", len(files))

	for i, file := range files {
		fmt.Printf("\n[%d/%d] 업스케일 중: %s\n", i+1, len(files), filepath.Base(file))

		// 출력 경로 생성
		relPath, _ := filepath.Rel(inputDir, file)
		outPath := filepath.Join(outputDir, relPath)
		outDir := filepath.Dir(outPath)
		os.MkdirAll(outDir, 0755)

		// 업스케일 시도 (API 먼저, 실패시 로컬)
		if err := u.upscaleWithAPI(file, outPath); err != nil {
			fmt.Printf("  ⚠️ API 업스케일 실패, 로컬 처리: %v\n", err)
			if err := u.upscaleLocal(file, outPath); err != nil {
				fmt.Printf("  ❌ 업스케일 실패: %v\n", err)
				continue
			}
		}

		fmt.Printf("  ✅ 저장됨: %s\n", outPath)

		// JSON 메타데이터 복사
		jsonSrc := strings.TrimSuffix(file, filepath.Ext(file)) + ".json"
		jsonDst := strings.TrimSuffix(outPath, filepath.Ext(outPath)) + ".json"
		copyFile(jsonSrc, jsonDst)
	}

	return nil
}

// upscaleWithAPI 무료 API로 업스케일 (ClipDrop/기타)
func (u *Upscaler) upscaleWithAPI(inputPath, outputPath string) error {
	// 무료 업스케일 API 사용 시도
	// 여러 무료 서비스 중 작동하는 것 사용

	// 방법 1: 로컬 고품질 리사이즈로 대체
	return fmt.Errorf("API 미사용, 로컬 처리")
}

// upscaleLocal 로컬에서 고품질 리사이즈
func (u *Upscaler) upscaleLocal(inputPath, outputPath string) error {
	// 원본 이미지 읽기
	file, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	// 새 크기 계산
	bounds := img.Bounds()
	newWidth := bounds.Dx() * u.scale
	newHeight := bounds.Dy() * u.scale

	// 고품질 리사이즈 (CatmullRom - 가장 좋은 품질)
	newImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.CatmullRom.Scale(newImg, newImg.Bounds(), img, bounds, draw.Over, nil)

	// 저장
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	return png.Encode(outFile, newImg)
}

// UpscaleWithDeepAI DeepAI API 사용 (무료 tier)
func (u *Upscaler) UpscaleWithDeepAI(inputPath, outputPath, apiKey string) error {
	// 이미지를 base64로 인코딩
	imgData, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	// multipart form 생성
	body := &bytes.Buffer{}
	body.WriteString("--boundary\r\n")
	body.WriteString("Content-Disposition: form-data; name=\"image\"; filename=\"image.png\"\r\n")
	body.WriteString("Content-Type: image/png\r\n\r\n")
	body.Write(imgData)
	body.WriteString("\r\n--boundary--\r\n")

	req, err := http.NewRequest("POST", "https://api.deepai.org/api/torch-srgan", body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	req.Header.Set("api-key", apiKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		OutputURL string `json:"output_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	// 결과 이미지 다운로드
	imgResp, err := http.Get(result.OutputURL)
	if err != nil {
		return err
	}
	defer imgResp.Body.Close()

	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, imgResp.Body)
	return err
}

// findImageFiles 이미지 파일 찾기
func findImageFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".png" || ext == ".jpg" || ext == ".jpeg" {
				files = append(files, path)
			}
		}
		return nil
	})
	return files, err
}

// copyFile 파일 복사
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// base64로 인코딩
func encodeToBase64(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// ConvertToJPEG PNG를 JPEG로 변환
func ConvertToJPEG(inputDir, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	files, err := findImageFiles(inputDir)
	if err != nil {
		return err
	}

	fmt.Printf("🔍 %d개 이미지 발견\n", len(files))

	for i, file := range files {
		fmt.Printf("[%d/%d] 변환 중: %s\n", i+1, len(files), filepath.Base(file))

		// 출력 경로 생성
		relPath, _ := filepath.Rel(inputDir, file)
		// 확장자를 .jpg로 변경
		outPath := filepath.Join(outputDir, strings.TrimSuffix(relPath, filepath.Ext(relPath))+".jpg")
		outDir := filepath.Dir(outPath)
		os.MkdirAll(outDir, 0755)

		// 이미지 변환
		if err := convertPNGtoJPEG(file, outPath); err != nil {
			fmt.Printf("  ❌ 변환 실패: %v\n", err)
			continue
		}
		fmt.Printf("  ✅ 저장됨: %s\n", outPath)

		// JSON 메타데이터 복사
		jsonSrc := strings.TrimSuffix(file, filepath.Ext(file)) + ".json"
		jsonDst := strings.TrimSuffix(outPath, filepath.Ext(outPath)) + ".json"
		copyFile(jsonSrc, jsonDst)
	}

	return nil
}

// convertPNGtoJPEG PNG를 JPEG로 변환
func convertPNGtoJPEG(inputPath, outputPath string) error {
	file, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// 고품질 JPEG로 저장 (품질 95%)
	return jpeg.Encode(outFile, img, &jpeg.Options{Quality: 95})
}

