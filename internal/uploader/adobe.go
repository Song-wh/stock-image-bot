package uploader

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// AdobeUploader Adobe Stock 업로더
type AdobeUploader struct {
	browser  *rod.Browser
	page     *rod.Page
	email    string
	password string
}

// ImageMetadata 이미지 메타데이터
type ImageMetadata struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
	Category    string   `json:"category"`
}

// NewAdobeUploader 새 Adobe 업로더 생성
func NewAdobeUploader(email, password string) *AdobeUploader {
	return &AdobeUploader{
		email:    email,
		password: password,
	}
}

// Connect 브라우저 연결
func (a *AdobeUploader) Connect() error {
	path, _ := launcher.LookPath()
	u := launcher.New().
		Bin(path).
		Leakless(false).
		Headless(false).
		MustLaunch()

	a.browser = rod.New().ControlURL(u).MustConnect()
	return nil
}

// Close 브라우저 종료
func (a *AdobeUploader) Close() {
	if a.browser != nil {
		a.browser.MustClose()
	}
}

// Login Adobe 계정 로그인
func (a *AdobeUploader) Login() error {
	fmt.Println("🔐 Adobe 로그인 중...")

	a.page = a.browser.MustPage("https://contributor.stock.adobe.com/")
	a.page.MustWaitLoad()
	time.Sleep(3 * time.Second)

	// "Link my Adobe ID" 또는 "Sign in" 버튼 클릭
	fmt.Println("  🔍 로그인 버튼 찾는 중...")

	// 방법 1: Link my Adobe ID 버튼
	linkBtn := a.page.MustElement("a.spectrum-Button--cta, a[href*='adobe'], button.spectrum-Button--cta")
	if linkBtn != nil {
		linkBtn.MustClick()
		fmt.Println("  ✅ 로그인 버튼 클릭")
	}

	time.Sleep(3 * time.Second)
	a.page.MustWaitLoad()

	// 이메일 입력 페이지 대기
	fmt.Println("  ⏳ 로그인 페이지 로딩...")
	time.Sleep(3 * time.Second)

	// 이메일 입력
	fmt.Println("  📧 이메일 입력 중...")
	a.page.MustElement("input[name='username'], input[type='email'], #EmailPage-EmailField").MustInput(a.email)
	time.Sleep(1 * time.Second)

	// Continue 버튼 클릭
	a.page.MustElement("button[type='submit'], button[data-id='EmailPage-ContinueButton']").MustClick()
	fmt.Println("  ✅ 이메일 제출")
	time.Sleep(3 * time.Second)

	// 비밀번호 입력
	fmt.Println("  🔑 비밀번호 입력 중...")
	a.page.MustElement("input[name='password'], input[type='password']").MustInput(a.password)
	time.Sleep(1 * time.Second)

	// 로그인 버튼 클릭
	a.page.MustElement("button[type='submit'], button[data-id='PasswordPage-ContinueButton']").MustClick()
	fmt.Println("  ✅ 로그인 제출")
	time.Sleep(5 * time.Second)

	fmt.Println("✅ Adobe Stock 로그인 완료!")
	return nil
}

// UploadImages 이미지 업로드
func (a *AdobeUploader) UploadImages(imageDir string) error {
	fmt.Println("\n📤 이미지 업로드 시작...")

	// 업로드 페이지로 이동
	a.page.MustNavigate("https://contributor.stock.adobe.com/ko/uploads")
	a.page.MustWaitLoad()
	time.Sleep(5 * time.Second)

	// 이미지 파일 찾기
	files, err := findJPEGImages(imageDir)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("업로드할 이미지가 없습니다")
	}

	fmt.Printf("📂 %d개 이미지 발견\n", len(files))

	// 파일 업로드
	fmt.Println("  📤 파일 업로드 중...")
	uploadInput := a.page.MustElement("input[type='file']")
	uploadInput.MustSetFiles(files...)

	// 업로드 완료 대기
	waitTime := time.Duration(len(files)*15) * time.Second
	fmt.Printf("  ⏳ 업로드 대기 중... (최대 %v)\n", waitTime)
	time.Sleep(waitTime)

	fmt.Println("\n✅ 업로드 완료!")
	fmt.Println("💡 브라우저에서 메타데이터 확인 후 'Submit' 버튼을 클릭하세요.")
	fmt.Println("\n⏳ 60초 후 브라우저가 닫힙니다. 그 전에 제출하세요!")
	time.Sleep(60 * time.Second)

	return nil
}

// findJPEGImages JPEG 이미지 찾기
func findJPEGImages(dir string) ([]string, error) {
	var files []string

	// _upscaled_jpeg 폴더 우선 확인
	jpegDir := strings.TrimSuffix(dir, "/") + "_upscaled_jpeg"
	if _, err := os.Stat(jpegDir); err == nil {
		dir = jpegDir
	} else {
		// _jpeg 폴더 확인
		jpegDir = dir + "_jpeg"
		if _, err := os.Stat(jpegDir); err == nil {
			dir = jpegDir
		}
	}

	fmt.Printf("  📂 이미지 폴더: %s\n", dir)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".jpg" || ext == ".jpeg" {
				files = append(files, path)
			}
		}
		return nil
	})
	return files, err
}

func findImages(dir string) ([]string, error) {
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

func loadMetadata(imagePath string) (*ImageMetadata, error) {
	base := strings.TrimSuffix(imagePath, filepath.Ext(imagePath))
	metaPath := base + ".json"

	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}

	var meta ImageMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func defaultMeta(imagePath string) *ImageMetadata {
	name := filepath.Base(imagePath)
	return &ImageMetadata{
		Title:       name,
		Description: "Stock photo",
		Keywords:    []string{"stock", "photo", "image"},
	}
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
