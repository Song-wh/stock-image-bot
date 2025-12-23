package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 설정
type Config struct {
	OpenAI struct {
		APIKey string `yaml:"api_key"`
	} `yaml:"openai"`
	Image struct {
		Model   string `yaml:"model"`
		Size    string `yaml:"size"`
		Quality string `yaml:"quality"`
		Style   string `yaml:"style"`
	} `yaml:"image"`
	Output struct {
		Directory string `yaml:"directory"`
	} `yaml:"output"`
	Categories []Category `yaml:"categories"`
}

// Category 카테고리
type Category struct {
	Name    string   `yaml:"name"`
	Count   int      `yaml:"count"`
	Prompts []string `yaml:"prompts"`
}

// ImageMetadata 메타데이터
type ImageMetadata struct {
	Prompt      string   `json:"prompt"`
	Category    string   `json:"category"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
	GeneratedAt string   `json:"generated_at"`
}

// OpenAI API 응답
type ImageResponse struct {
	Data []struct {
		URL string `json:"url"`
	} `json:"data"`
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func main() {
	fmt.Println("🖼️  AI 스톡 이미지 생성 봇")
	fmt.Println("══════════════════════════════════════════")

	// 설정 로드
	config, err := loadConfig("config.yaml")
	if err != nil {
		fmt.Printf("❌ 설정 로드 실패: %v\n", err)
		os.Exit(1)
	}

	if config.OpenAI.APIKey == "" {
		fmt.Println("❌ OpenAI API 키가 설정되지 않았습니다.")
		os.Exit(1)
	}
	fmt.Println("✅ 설정 로드 완료")

	// 출력 디렉토리 생성
	outputDir := config.Output.Directory
	if outputDir == "" {
		outputDir = "./generated_images"
	}
	os.MkdirAll(outputDir, 0755)
	fmt.Printf("📂 출력 폴더: %s\n", outputDir)

	// 테스트 모드 확인
	testMode := len(os.Args) > 1 && os.Args[1] == "--test"
	if testMode {
		fmt.Println("\n🧪 테스트 모드: 1장만 생성합니다")
	}

	// 이미지 생성
	totalGenerated := 0
	totalCost := 0.0

	for _, category := range config.Categories {
		if testMode && totalGenerated >= 1 {
			break
		}

		count := category.Count
		if testMode {
			count = 1
		}

		generated := generateCategoryImages(config, category, count, outputDir)
		totalGenerated += generated

		// 비용 계산
		costPerImage := 0.04
		if config.Image.Quality == "hd" {
			costPerImage = 0.08
		}
		totalCost += float64(generated) * costPerImage
	}

	// 결과 출력
	fmt.Println("\n══════════════════════════════════════════")
	fmt.Println("📊 생성 결과")
	fmt.Println("══════════════════════════════════════════")
	fmt.Printf("  ✅ 생성된 이미지: %d장\n", totalGenerated)
	fmt.Printf("  💰 예상 비용: $%.2f (약 %d원)\n", totalCost, int(totalCost*1300))
	fmt.Printf("  📂 저장 위치: %s\n", outputDir)
	fmt.Println("\n✨ 완료!")
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func generateCategoryImages(config *Config, category Category, count int, outputDir string) int {
	fmt.Printf("\n📁 카테고리: %s (%d장)\n", category.Name, count)
	fmt.Println("────────────────────────────────────────")

	// 카테고리 폴더 생성
	catDir := filepath.Join(outputDir, category.Name)
	os.MkdirAll(catDir, 0755)

	generated := 0
	prompts := category.Prompts
	if len(prompts) < count {
		count = len(prompts)
	}

	for i := 0; i < count; i++ {
		prompt := prompts[i]
		fmt.Printf("\n  [%d/%d] 생성 중...\n", i+1, count)
		fmt.Printf("  📝 프롬프트: %s...\n", truncate(prompt, 50))

		// 이미지 생성
		imageURL, err := generateImage(config, prompt)
		if err != nil {
			fmt.Printf("  ❌ 이미지 생성 실패: %v\n", err)
			continue
		}
		fmt.Println("  ✅ 이미지 생성 완료")

		// 파일명 생성
		timestamp := time.Now().Format("20060102_150405")
		filename := fmt.Sprintf("%s_%s_%d.png", category.Name, timestamp, i+1)
		savePath := filepath.Join(catDir, filename)

		// 다운로드
		if err := downloadImage(imageURL, savePath); err != nil {
			fmt.Printf("  ❌ 다운로드 실패: %v\n", err)
			continue
		}
		fmt.Printf("  💾 저장됨: %s\n", savePath)
		generated++

		// 메타데이터 생성
		fmt.Println("  📋 메타데이터 생성 중...")
		metadata := generateMetadata(config, prompt, category.Name)

		// 메타데이터 저장
		metaPath := savePath[:len(savePath)-4] + ".json"
		saveMetadata(metadata, metaPath)
		fmt.Printf("  📋 메타데이터 저장됨: %s\n", metaPath)

		// API 속도 제한 방지
		time.Sleep(2 * time.Second)
	}

	return generated
}

func generateImage(config *Config, prompt string) (string, error) {
	url := "https://api.openai.com/v1/images/generations"

	requestBody := map[string]interface{}{
		"model":   config.Image.Model,
		"prompt":  prompt,
		"size":    config.Image.Size,
		"quality": config.Image.Quality,
		"style":   config.Image.Style,
		"n":       1,
	}

	jsonBody, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.OpenAI.APIKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API 오류 (%d): %s", resp.StatusCode, string(body))
	}

	var result ImageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.Data) == 0 {
		return "", fmt.Errorf("이미지 URL 없음")
	}

	return result.Data[0].URL, nil
}

func downloadImage(url, savePath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	file, err := os.Create(savePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

func generateMetadata(config *Config, prompt, category string) *ImageMetadata {
	url := "https://api.openai.com/v1/chat/completions"

	systemPrompt := `You are a stock image metadata expert. Generate metadata for stock image sites.
Return JSON with: title (max 200 chars), description (max 500 chars), keywords (array of 30-50 terms).
Focus on commercial search terms.`

	userPrompt := fmt.Sprintf("Category: %s\nImage prompt: %s\n\nGenerate metadata JSON.", category, prompt)

	requestBody := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"response_format": map[string]string{"type": "json_object"},
	}

	jsonBody, _ := json.Marshal(requestBody)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.OpenAI.APIKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return defaultMetadata(prompt, category)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result ChatResponse
	if err := json.Unmarshal(body, &result); err != nil || len(result.Choices) == 0 {
		return defaultMetadata(prompt, category)
	}

	var metadata ImageMetadata
	if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &metadata); err != nil {
		return defaultMetadata(prompt, category)
	}

	metadata.Prompt = prompt
	metadata.Category = category
	metadata.GeneratedAt = time.Now().Format(time.RFC3339)

	return &metadata
}

func defaultMetadata(prompt, category string) *ImageMetadata {
	return &ImageMetadata{
		Prompt:      prompt,
		Category:    category,
		Title:       truncate(prompt, 200),
		Description: prompt,
		Keywords:    []string{category, "stock photo", "commercial use", "high quality"},
		GeneratedAt: time.Now().Format(time.RFC3339),
	}
}

func saveMetadata(metadata *ImageMetadata, path string) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

