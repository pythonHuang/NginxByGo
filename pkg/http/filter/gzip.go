package filter

import (
	"compress/gzip"
	"io"
	"strings"
	"github.com/nginxgo/nginxgo/pkg/http"
)

// GzipFilter Gzip 压缩过滤器
type GzipFilter struct {
	Level   int    // 压缩级别 1-9
	MinSize int    // 最小压缩大小
}

// NewGzipFilter 创建 Gzip 过滤器
func NewGzipFilter(level int) *GzipFilter {
	if level < 1 {
		level = 1
	}
	if level > 9 {
		level = 9
	}
	return &GzipFilter{
		Level:   level,
		MinSize: 1024, // 至少 1KB 才压缩
	}
}

// Filter 应用 gzip 压缩
func (f *GzipFilter) Filter(w http.ResponseWriter, r *http.Request) {
	// 检查是否支持 gzip
	acceptEncoding := r.GetHeader("Accept-Encoding")
	if !strings.Contains(acceptEncoding, "gzip") {
		return
	}

	// 检查内容类型是否应该压缩
	contentType := w.(*http.Response).Headers["Content-Type"]
	if !shouldCompress(contentType) {
		return
	}

	// 获取响应体（需要先获取内容）
	// 注意: 这是一个简化的实现
	// 实际生产环境应该使用更复杂的过滤链
}

// shouldCompress 检查内容类型是否应该压缩
func shouldCompress(contentType string) bool {
	compressibleTypes := []string{
		"text/html",
		"text/css",
		"text/javascript",
		"text/xml",
		"text/plain",
		"application/json",
		"application/javascript",
		"application/xml",
		"application/css",
	}

	for _, t := range compressibleTypes {
		if strings.Contains(contentType, t) {
			return true
		}
	}
	return false
}

// GzipWriter 包装ResponseWriter以支持gzip压缩
type GzipWriter struct {
	http.ResponseWriter
	gzipWriter *gzip.Writer
}

// Write 写入数据
func (g *GzipWriter) Write(b []byte) (int, error) {
	if g.gzipWriter == nil {
		g.gzipWriter = gzip.NewWriter(g.ResponseWriter)
		g.ResponseWriter.SetHeader("Content-Encoding", "gzip")
	}
	return g.gzipWriter.Write(b)
}

// Flush 刷新
func (g *GzipWriter) Flush() {
	if g.gzipWriter != nil {
		g.gzipWriter.Flush()
	}
}

// Close 关闭
func (g *GzipWriter) Close() error {
	if g.gzipWriter != nil {
		return g.gzipWriter.Close()
	}
	return nil
}

// NewGzipResponseWriter 创建支持gzip的响应写入器
func NewGzipResponseWriter(w http.ResponseWriter) *GzipWriter {
	// 检查是否应该启用 gzip
	// 实际实现需要检查请求头和响应内容
	return &GzipWriter{
		ResponseWriter: w,
	}
}