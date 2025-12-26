//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Meta struct {
	Title    string   `json:"title"`
	Keywords []string `json:"keywords"`
}

func main() {
	dir := "./generated_images_upscaled_jpeg"
	
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println("📋 Freepik 메타데이터 복사용")
	fmt.Println("=" + strings.Repeat("=", 79))
	
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".json") {
			data, _ := os.ReadFile(path)
			var meta Meta
			json.Unmarshal(data, &meta)
			
			imgPath := strings.TrimSuffix(path, ".json") + ".jpg"
			imgName := filepath.Base(imgPath)
			
			fmt.Printf("\n📸 %s\n", imgName)
			fmt.Println("-" + strings.Repeat("-", 79))
			fmt.Printf("Title: %s\n", meta.Title)
			fmt.Printf("\nKeywords (복사용):\n%s\n", strings.Join(meta.Keywords, ", "))
		}
		return nil
	})
}
