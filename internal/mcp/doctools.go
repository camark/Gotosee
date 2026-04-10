// Package mcp 提供 MCP 服务器功能。
package mcp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ============================================================================
// PDF 工具
// ============================================================================

// PDFOperation PDF 操作类型。
type PDFOperation string

const (
	PDFExtractText  PDFOperation = "extract_text"
	PDFExtractImages PDFOperation = "extract_images"
	PDFMetadata    PDFOperation = "metadata"
)

// PDFToolParams PDF 工具参数。
type PDFToolParams struct {
	Path      string         `json:"path"`
	Operation PDFOperation   `json:"operation"`
}

// PDFTool 处理 PDF 文件。
func PDFTool(params PDFToolParams, cacheDir string) (*ToolResult, error) {
	// 检查文件是否存在
	if _, err := os.Stat(params.Path); os.IsNotExist(err) {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("PDF file not found: %s", params.Path)}},
		}, nil
	}

	switch params.Operation {
	case PDFExtractText:
		return extractPDFText(params.Path)
	case PDFExtractImages:
		return extractPDFImages(params.Path, cacheDir)
	case PDFMetadata:
		return extractPDFMetadata(params.Path)
	default:
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("Unknown PDF operation: %s", params.Operation)}},
		}, nil
	}
}

// extractPDFText 提取 PDF 文本。
// 注意：完整的 PDF 解析需要第三方库，这里提供简化版本。
func extractPDFText(path string) (*ToolResult, error) {
	// TODO: 实现完整的 PDF 文本提取
	// 可以使用 pdfcpu 或 其他 Go PDF 库
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// 简单的 PDF 文本提取（仅适用于简单 PDF）
	text := extractTextFromPDF(data)

	return &ToolResult{
		Content: []Content{
			{Type: "text", Text: fmt.Sprintf("Extracted text from PDF:\n\n%s", text)},
		},
	}, nil
}

// extractTextFromPDF 从 PDF 数据中提取文本（简化实现）。
func extractTextFromPDF(data []byte) string {
	// 这是一个简化实现，实际应该使用 PDF 解析库
	// PDF 文件以 %PDF- 开头
	content := string(data)

	// 查找文本流
	var text strings.Builder
	lines := strings.Split(content, "\n")
	inTextObject := false

	for _, line := range lines {
		if strings.Contains(line, "BT") { // Begin Text
			inTextObject = true
			continue
		}
		if strings.Contains(line, "ET") { // End Text
			inTextObject = false
			continue
		}
		if inTextObject && strings.Contains(line, "Tj") {
			// 提取 Tj 操作符的文本
			if start := strings.Index(line, "("); start != -1 {
				if end := strings.Index(line[start:], ")"); end != -1 {
					text.WriteString(line[start+1 : start+end])
					text.WriteString("\n")
				}
			}
		}
	}

	return text.String()
}

// extractPDFImages 提取 PDF 图片。
func extractPDFImages(path, cacheDir string) (*ToolResult, error) {
	// TODO: 实现 PDF 图片提取
	return &ToolResult{
		Content: []Content{
			{Type: "text", Text: "PDF image extraction not yet implemented"},
		},
	}, nil
}

// extractPDFMetadata 提取 PDF 元数据。
func extractPDFMetadata(path string) (*ToolResult, error) {
	// TODO: 实现 PDF 元数据提取
	return &ToolResult{
		Content: []Content{
			{Type: "text", Text: "PDF metadata extraction not yet implemented"},
		},
	}, nil
}

// ============================================================================
// DOCX 工具
// ============================================================================

// DOCXOperation DOCX 操作类型。
type DOCXOperation string

const (
	DOCXExtractText DOCXOperation = "extract_text"
	DOCXMetadata    DOCXOperation = "metadata"
	DOCXCountWords  DOCXOperation = "count_words"
)

// DOCXToolParams DOCX 工具参数。
type DOCXToolParams struct {
	Path      string         `json:"path"`
	Operation DOCXOperation  `json:"operation"`
}

// DOCXTool 处理 DOCX 文件。
func DOCXTool(params DOCXToolParams) (*ToolResult, error) {
	// 检查文件是否存在
	if _, err := os.Stat(params.Path); os.IsNotExist(err) {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("DOCX file not found: %s", params.Path)}},
		}, nil
	}

	switch params.Operation {
	case DOCXExtractText:
		return extractDOCXText(params.Path)
	case DOCXMetadata:
		return extractDOCXMetadata(params.Path)
	case DOCXCountWords:
		return countDOCXWords(params.Path)
	default:
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("Unknown DOCX operation: %s", params.Operation)}},
		}, nil
	}
}

// extractDOCXText 提取 DOCX 文本。
// DOCX 是 ZIP 压缩包，包含 XML 文件。
func extractDOCXText(path string) (*ToolResult, error) {
	// TODO: 使用 zip 包解压并解析 DOCX
	// DOCX 结构：word/document.xml 包含主要文本

	return &ToolResult{
		Content: []Content{
			{Type: "text", Text: "DOCX text extraction not yet implemented"},
		},
	}, nil
}

// extractDOCXMetadata 提取 DOCX 元数据。
func extractDOCXMetadata(path string) (*ToolResult, error) {
	// TODO: 提取 DOCX 元数据
	return &ToolResult{
		Content: []Content{
			{Type: "text", Text: "DOCX metadata extraction not yet implemented"},
		},
	}, nil
}

// countDOCXWords 统计 DOCX 字数。
func countDOCXWords(path string) (*ToolResult, error) {
	// TODO: 统计 DOCX 字数
	return &ToolResult{
		Content: []Content{
			{Type: "text", Text: "DOCX word count not yet implemented"},
		},
	}, nil
}

// ============================================================================
// XLSX 工具
// ============================================================================

// XLSXOperation XLSX 操作类型。
type XLSXOperation string

const (
	XLSXReadSheet    XLSXOperation = "read_sheet"
	XLSXListSheets   XLSXOperation = "list_sheets"
	XLSXExtractCell  XLSXOperation = "extract_cell"
)

// XLSXToolParams XLSX 工具参数。
type XLSXToolParams struct {
	Path      string         `json:"path"`
	Operation XLSXOperation  `json:"operation"`
	Sheet     string         `json:"sheet,omitempty"`
	Cell      string         `json:"cell,omitempty"` // e.g., "A1"
	Range     string         `json:"range,omitempty"` // e.g., "A1:B10"
}

// XLSXTool 处理 XLSX 文件。
func XLSXTool(params XLSXToolParams) (*ToolResult, error) {
	// 检查文件是否存在
	if _, err := os.Stat(params.Path); os.IsNotExist(err) {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("XLSX file not found: %s", params.Path)}},
		}, nil
	}

	switch params.Operation {
	case XLSXListSheets:
		return listXLSXSheets(params.Path)
	case XLSXReadSheet:
		return readXLSXSheet(params.Path, params.Sheet, params.Range)
	case XLSXExtractCell:
		return extractXLSXCell(params.Path, params.Sheet, params.Cell)
	default:
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("Unknown XLSX operation: %s", params.Operation)}},
		}, nil
	}
}

// listXLSXSheets 列出 XLSX 工作表。
func listXLSXSheets(path string) (*ToolResult, error) {
	// TODO: 使用 zip 包解析 XLSX 并列出工作表
	return &ToolResult{
		Content: []Content{
			{Type: "text", Text: "XLSX list sheets not yet implemented"},
		},
	}, nil
}

// readXLSXSheet 读取 XLSX 工作表。
func readXLSXSheet(path, sheet, rangeStr string) (*ToolResult, error) {
	// TODO: 读取 XLSX 工作表数据
	return &ToolResult{
		Content: []Content{
			{Type: "text", Text: "XLSX read sheet not yet implemented"},
		},
	}, nil
}

// extractXLSXCell 提取 XLSX 单元格。
func extractXLSXCell(path, sheet, cell string) (*ToolResult, error) {
	// TODO: 提取 XLSX 单元格数据
	return &ToolResult{
		Content: []Content{
			{Type: "text", Text: "XLSX extract cell not yet implemented"},
		},
	}, nil
}

// ============================================================================
// Web Scraping 工具
// ============================================================================

// WebScrapeParams 网页抓取参数。
type WebScrapeParams struct {
	URL    string `json:"url"`
	SaveAs string `json:"save_as,omitempty"` // "text", "json", "binary"
}

// WebScrapeTool 抓取网页内容。
func WebScrapeTool(params WebScrapeParams) (*ToolResult, error) {
	// TODO: 实现网页抓取
	return &ToolResult{
		Content: []Content{
			{Type: "text", Text: "Web scraping not yet implemented"},
		},
	}, nil
}

// ============================================================================
// Cache 工具
// ============================================================================

// CacheCommand 缓存命令。
type CacheCommand string

const (
	CacheList   CacheCommand = "list"
	CacheView   CacheCommand = "view"
	CacheDelete CacheCommand = "delete"
	CacheClear  CacheCommand = "clear"
)

// CacheToolParams 缓存工具参数。
type CacheToolParams struct {
	Command CacheCommand `json:"command"`
	Key     string       `json:"key,omitempty"`
}

// CacheTool 管理缓存。
func CacheTool(params CacheToolParams, cacheDir string) (*ToolResult, error) {
	switch params.Command {
	case CacheList:
		return listCache(cacheDir)
	case CacheView:
		return viewCache(params.Key, cacheDir)
	case CacheDelete:
		return deleteCache(params.Key, cacheDir)
	case CacheClear:
		return clearCache(cacheDir)
	default:
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("Unknown cache command: %s", params.Command)}},
		}, nil
	}
}

// listCache 列出缓存文件。
func listCache(cacheDir string) (*ToolResult, error) {
	files, err := os.ReadDir(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return &ToolResult{
				Content: []Content{{Type: "text", Text: "Cache directory is empty"}},
			}, nil
		}
		return nil, err
	}

	var sb strings.Builder
	sb.WriteString("Cached files:\n")
	for _, f := range files {
		if !f.IsDir() {
			sb.WriteString(fmt.Sprintf("  - %s\n", f.Name()))
		}
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
	}, nil
}

// viewCache 查看缓存文件内容。
func viewCache(key, cacheDir string) (*ToolResult, error) {
	path := filepath.Join(cacheDir, key)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ToolResult{
				IsError: true,
				Content: []Content{{Type: "text", Text: fmt.Sprintf("Cache key not found: %s", key)}},
			}, nil
		}
		return nil, err
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: string(data)}},
	}, nil
}

// deleteCache 删除缓存文件。
func deleteCache(key, cacheDir string) (*ToolResult, error) {
	path := filepath.Join(cacheDir, key)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return &ToolResult{
				IsError: true,
				Content: []Content{{Type: "text", Text: fmt.Sprintf("Cache key not found: %s", key)}},
			}, nil
		}
		return nil, err
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("Deleted cache key: %s", key)}},
	}, nil
}

// clearCache 清空缓存。
func clearCache(cacheDir string) (*ToolResult, error) {
	err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return os.Remove(path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: "Cache cleared"}},
	}, nil
}

// ============================================================================
// File Operations
// ============================================================================

// CopyFile 复制文件。
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
