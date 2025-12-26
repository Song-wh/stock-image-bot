package uploader

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// FreepikUploader Freepik 업로더
type FreepikUploader struct {
	browser  *rod.Browser
	email    string
	password string
}

// NewFreepikUploader 새 Freepik 업로더 생성
func NewFreepikUploader(email, password string) *FreepikUploader {
	return &FreepikUploader{
		email:    email,
		password: password,
	}
}

// Connect 브라우저 연결
func (f *FreepikUploader) Connect() error {
	path, _ := launcher.LookPath()
	u := launcher.New().
		Bin(path).
		Leakless(false).
		Headless(false). // 디버깅용
		MustLaunch()

	f.browser = rod.New().ControlURL(u).MustConnect()
	return nil
}

// Close 브라우저 종료
func (f *FreepikUploader) Close() {
	if f.browser != nil {
		f.browser.MustClose()
	}
}

// Login Freepik 로그인
func (f *FreepikUploader) Login() error {
	page := f.browser.MustPage("https://contributor.freepik.com/login")
	page.MustWaitLoad()
	time.Sleep(3 * time.Second)

	// 쿠키 수락 (있으면)
	if acceptBtn, err := page.Element("button[id='onetrust-accept-btn-handler']"); err == nil && acceptBtn != nil {
		acceptBtn.MustClick()
		time.Sleep(1 * time.Second)
	}

	// 이메일 입력
	emailInput := page.MustElement("input[name='email']")
	emailInput.MustInput(f.email)

	// 비밀번호 입력
	pwInput := page.MustElement("input[name='password']")
	pwInput.MustInput(f.password)

	// 로그인 버튼
	page.MustElement("button[type='submit']").MustClick()
	time.Sleep(5 * time.Second)

	fmt.Println("✅ Freepik 로그인 완료")
	return nil
}

// UploadImages 이미지 업로드
func (f *FreepikUploader) UploadImages(imageDir string) error {
	page := f.browser.MustPage("https://contributor.freepik.com/panel/upload")
	page.MustWaitLoad()
	time.Sleep(3 * time.Second)

	// 이미지 파일 찾기
	files, err := findImages(imageDir)
	if err != nil {
		return err
	}

	fmt.Printf("📤 Freepik: %d개 이미지 업로드 시작\n", len(files))

	// 드래그 앤 드롭 영역 찾기
	uploadInput := page.MustElement("input[type='file']")
	
	// 모든 파일 한번에 업로드
	uploadInput.MustSetFiles(files...)
	
	fmt.Println("  ⏳ 업로드 중...")
	time.Sleep(time.Duration(len(files)*5) * time.Second) // 파일당 5초

	// 각 이미지에 메타데이터 입력
	for i, file := range files {
		fmt.Printf("\n[%d/%d] 메타데이터 입력: %s\n", i+1, len(files), filepath.Base(file))

		meta, err := loadMetadata(file)
		if err != nil {
			meta = defaultMeta(file)
		}

		// 이미지 항목 찾기
		items := page.MustElements(".upload-item, .file-item")
		if i < len(items) {
			items[i].MustClick()
			time.Sleep(1 * time.Second)

			// 제목
			if titleInput, err := page.Element("input[name='title']"); err == nil && titleInput != nil {
				titleInput.MustSelectAllText().MustInput(meta.Title)
			}

			// 태그/키워드
			if tagsInput, err := page.Element("input[name='tags']"); err == nil && tagsInput != nil {
				for _, kw := range meta.Keywords[:min(50, len(meta.Keywords))] {
					tagsInput.MustInput(kw + ",")
				}
			}

			time.Sleep(500 * time.Millisecond)
		}
	}

	// 저장/제출
	if submitBtn, err := page.Element("button[type='submit'], .submit-btn"); err == nil && submitBtn != nil {
		submitBtn.MustClick()
		time.Sleep(3 * time.Second)
	}

	fmt.Println("\n✅ Freepik 업로드 완료")
	return nil
}

// CheckContributorStatus 컨트리뷰터 상태 확인
func (f *FreepikUploader) CheckContributorStatus() (string, error) {
	page := f.browser.MustPage("https://contributor.freepik.com/panel")
	page.MustWaitLoad()
	time.Sleep(3 * time.Second)

	// 상태 확인
	status := "unknown"
	
	// 승인 대기, 승인됨 등 상태 확인
	if statusEl, err := page.Element(".contributor-status, .status-badge"); err == nil && statusEl != nil {
		text, _ := statusEl.Text()
		status = text
	}

	return status, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}


