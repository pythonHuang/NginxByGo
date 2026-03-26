package handler

import (
	"net/http"
	"path"
	"os"
	"strings"
	"mime"
	"github.com/nginxgo/nginxgo/pkg/http"
)

// StaticHandler 静态文件处理器
type StaticHandler struct {
	Root     string      // 文档根目录
	Index    []string    // 默认首页
}

// NewStaticHandler 创建静态文件处理器
func NewStaticHandler(root string, index []string) *StaticHandler {
	return &StaticHandler{
		Root:  root,
		Index: index,
	}
}

// ServeHTTP 处理静态文件请求
func (h *StaticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 构建文件路径
	filePath := path.Join(h.Root, r.Path)

	// 检查路径安全（防止目录遍历）
	if !strings.HasPrefix(filePath, h.Root) {
		w.SetStatus(403)
		w.WriteString("Forbidden")
		return
	}

	// 获取文件信息
	stat, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			w.SetStatus(404)
			w.WriteString("Not Found")
		} else {
			w.SetStatus(500)
			w.WriteString("Internal Server Error")
		}
		return
	}

	// 如果是目录，查找默认首页
	if stat.IsDir() {
		for _, indexFile := range h.Index {
			indexPath := path.Join(filePath, indexFile)
			if info, err := os.Stat(indexPath); err == nil && !info.IsDir() {
				filePath = indexPath
				stat = info
				break
			}
		}
	}

	// 检查是否是文件
	if stat.IsDir() {
		w.SetStatus(403)
		w.WriteString("Forbidden")
		return
	}

	// 设置响应头
	ext := path.Ext(filePath)
	w.SetContentType(mime.TypeByExtension(ext))
	w.SetContentLength(int(stat.Size()))

	// ETag
	etag := computeETag(stat)
	w.SetHeader("ETag", etag)

	// Last-Modified
	w.SetHeader("Last-Modified", stat.ModTime().Format(http.TimeFormat))

	// 读取并发送文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		w.SetStatus(500)
		return
	}

	w.Write(data)
}

// computeETag 计算 ETag
func computeETag(info os.FileInfo) string {
	return ""
}