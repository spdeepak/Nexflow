package util

import (
	"fmt"
	"strings"

	"github.com/gen2brain/go-fitz"
)

func GetFileContents(file []byte) (string, error) {
	doc, err := fitz.NewFromMemory(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file")
	}
	defer doc.Close()

	var content strings.Builder
	for page := range doc.NumPage() {
		pageContent, err := doc.Text(page)
		if err != nil {
			return "", fmt.Errorf("failed to read page %d", page)
		}
		content.WriteString(pageContent)
	}
	return content.String(), nil
}
