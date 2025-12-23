package uploader

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/input"
)

// AdobeUploader Adobe Stock 업로더
type AdobeUploader struct {
	browser *rod.Browser
	email   string
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
		Headless(false). // 디버깅용 - 나중에 true로 변경
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
	page := a.browser.MustPage("https://stock.adobe.com/contributor")
	page.MustWaitLoad()
	time.Sleep(3 * time.Second)

	// Sign In 버튼 클릭
	signInBtn := page.MustElement("a[data-t='header-sign-in']")
	if signInBtn != nil {
		signInBtn.MustClick()
		time.Sleep(2 * time.Second)
	}

	// 이메일 입력
	page.MustWaitLoad()
	time.Sleep(2 * time.Second)
	
	emailInput := page.MustElement("input[name='username']")
	emailInput.MustInput(a.email)
	
	// Continue 버튼
	page.MustElement("button[data-id='EmailPage-ContinueButton']").MustClick()
	time.Sleep(2 * time.Second)

	// 비밀번호 입력
	pwInput := page.MustElement("input[name='password']")
	pwInput.MustInput(a.password)

	// Login 버튼
	page.MustElement("button[data-id='PasswordPage-ContinueButton']").MustClick()
	time.Sleep(5 * time.Second)

	fmt.Println("✅ Adobe Stock 로그인 완료")
	return nil
}

// UploadImages 이미지 업로드
func (a *AdobeUploader) UploadImages(imageDir string) error {
	page := a.browser.MustPage("https://contributor.stock.adobe.com/uploads")
	page.MustWaitLoad()
	time.Sleep(3 * time.Second)

	// 이미지 파일 찾기
	files, err := findImages(imageDir)
	if err != nil {
		return err
	}

	fmt.Printf("📤 %d개 이미지 업로드 시작\n", len(files))

	for i, file := range files {
		fmt.Printf("\n[%d/%d] %s\n", i+1, len(files), filepath.Base(file))

		// 메타데이터 로드
		meta, err := loadMetadata(file)
		if err != nil {
			fmt.Printf("  ⚠️ 메타데이터 없음, 기본값 사용\n")
			meta = defaultMeta(file)
		}

		// 파일 업로드
		uploadInput := page.MustElement("input[type='file']")
		uploadInput.MustSetFiles(file)
		time.Sleep(5 * time.Second)

		// 업로드 완료 대기
		fmt.Println("  ⏳ 업로드 중...")
		time.Sleep(10 * time.Second)

		// 메타데이터 입력을 위해 이미지 클릭
		page.MustWaitLoad()
		
		// 최근 업로드된 이미지 선택
		imgs := page.MustElements(".upload-item")
		if len(imgs) > 0 {
			imgs[0].MustClick()
			time.Sleep(2 * time.Second)

			// 제목 입력
			if titleInput, err := page.Element("input[name='title']"); err == nil && titleInput != nil {
				titleInput.MustSelectAllText().MustInput(meta.Title)
			}

			// 설명 입력
			if descInput, err := page.Element("textarea[name='description']"); err == nil && descInput != nil {
				descInput.MustSelectAllText().MustInput(meta.Description)
			}

			// 키워드 입력
			if keywordInput, err := page.Element("input[name='keywords']"); err == nil && keywordInput != nil {
				for _, kw := range meta.Keywords {
					keywordInput.MustInput(kw)
					keywordInput.MustType(input.Comma)
					time.Sleep(100 * time.Millisecond)
				}
			}

			// 저장
			if saveBtn, err := page.Element("button[type='submit']"); err == nil && saveBtn != nil {
				saveBtn.MustClick()
				time.Sleep(2 * time.Second)
			}
		}

		fmt.Println("  ✅ 업로드 완료")
		time.Sleep(2 * time.Second)
	}

	return nil
}

// SubmitForReview 심사 제출
func (a *AdobeUploader) SubmitForReview() error {
	page := a.browser.MustPage("https://contributor.stock.adobe.com/uploads")
	page.MustWaitLoad()
	time.Sleep(3 * time.Second)

	// 모든 이미지 선택
	selectAll := page.MustElement("input[type='checkbox'][name='select-all']")
	if selectAll != nil {
		selectAll.MustClick()
		time.Sleep(1 * time.Second)
	}

	// Submit 버튼 클릭
	submitBtn := page.MustElement("button[data-action='submit']")
	if submitBtn != nil {
		submitBtn.MustClick()
		time.Sleep(2 * time.Second)
	}

	fmt.Println("✅ 심사 제출 완료")
	return nil
}

// 유틸리티 함수들
func findImages(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (filepath.Ext(path) == ".png" || filepath.Ext(path) == ".jpg") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func loadMetadata(imagePath string) (*ImageMetadata, error) {
	metaPath := imagePath[:len(imagePath)-4] + ".json"
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

