# 🖼️ AI Stock Image Bot

DALL-E 3를 이용한 스톡 이미지 자동 생성 봇

## ✨ 기능

- **DALL-E 3** 이미지 생성
- **GPT-4o-mini** 메타데이터 자동 생성 (제목, 설명, 키워드)
- 카테고리별 이미지 분류
- 스톡 사이트 업로드용 JSON 메타데이터

## 📦 설치

```bash
# 빌드
go build -o stock-image-bot.exe ./cmd/stock-image-bot/

# 설정 파일 복사
cp config.yaml.example config.yaml
# config.yaml에 OpenAI API 키 입력
```

## 🚀 사용법

```bash
# 전체 실행 (기본 20장)
./stock-image-bot.exe

# 테스트 (1장만)
./stock-image-bot.exe --test
```

## 💰 비용

| 품질 | 장당 가격 | 20장 |
|------|----------|------|
| standard | $0.04 (~52원) | $0.80 (~1,040원) |
| hd | $0.08 (~104원) | $1.60 (~2,080원) |

## 📁 출력 구조

```
generated_images/
├── business/
│   ├── business_20241223_143000_1.png
│   └── business_20241223_143000_1.json  # 메타데이터
├── technology/
├── lifestyle/
└── nature/
```

## 📋 메타데이터 예시

```json
{
  "title": "Professional Business Meeting in Modern Office",
  "description": "A diverse team collaborating in a sleek, modern office...",
  "keywords": ["business", "meeting", "teamwork", "office", "professional", ...]
}
```

## 🛒 판매 사이트

- [Adobe Stock](https://stock.adobe.com) - 33% 수익률
- [Shutterstock](https://www.shutterstock.com) - 15-40% 수익률
- [Freepik](https://www.freepik.com) - 30-50% 수익률, 진입장벽 낮음
- [iStock](https://www.istockphoto.com) - Getty 계열
- [Dreamstime](https://www.dreamstime.com) - 초보 친화적
