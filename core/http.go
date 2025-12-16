package core

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/tekintian/go-bbdown/util"
)

// HTTPClient HTTP客户端
type HTTPClient struct {
	Client *http.Client
}

// NewHTTPClient 创建新的HTTP客户端
func NewHTTPClient() *HTTPClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   2 * time.Minute,
	}

	return &HTTPClient{Client: client}
}

// GetWebSource 获取网页内容
func (h *HTTPClient) GetWebSource(urlStr, userAgent string) (string, error) {
	if userAgent == "" {
		userAgent = getRandomUserAgent()
	}

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	if strings.Contains(urlStr, "api.bilibili.com") {
		req.Header.Set("Referer", "https://www.bilibili.com/")
	}
	if strings.Contains(urlStr, "api.bilibili.tv") {
		req.Header.Set("sec-ch-ua", `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`)
	}
	if strings.Contains(urlStr, "space.bilibili.com") {
		req.Header.Set("Referer", "https://www.bilibili.com/")
	}

	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	resp, err := h.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 处理频率限制
		if resp.StatusCode == 429 {
			return "", fmt.Errorf("请求过于频繁，请稍后再试")
		}
		return "", fmt.Errorf("HTTP状态码: %d", resp.StatusCode)
	}

	// 处理gzip压缩
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", fmt.Errorf("解压失败: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	return string(body), nil
}

// GetWebLocation 获取重定向地址
func (h *HTTPClient) GetWebLocation(urlStr, userAgent string) (string, error) {
	if userAgent == "" {
		userAgent = getRandomUserAgent()
	}

	req, err := http.NewRequest("HEAD", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	resp, err := h.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 处理频率限制
		if resp.StatusCode == 429 {
			return "", fmt.Errorf("请求过于频繁，请稍后再试")
		}
		return "", fmt.Errorf("HTTP状态码: %d", resp.StatusCode)
	}

	finalURL := resp.Request.URL.String()
	return finalURL, nil
}

// PostRequest 发送POST请求
func (h *HTTPClient) PostRequest(urlStr string, data []byte, headers map[string]string) (string, error) {
	req, err := http.NewRequest("POST", urlStr, bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/grpc")
	req.Header.Set("Accept", "*/*")

	if headers != nil {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	} else {
		// 默认APP请求头
		req.Header.Set("User-Agent", "Dalvik/2.1.0 (Linux; U; Android 6.0.1; oneplus a5010 Build/V417IR) 6.10.0 os/android model/oneplus a5010 mobi_app/android build/6100500 channel/bili innerVer/6100500 osVer/6.0.1 network/2")
		req.Header.Set("grpc-encoding", "gzip")
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 处理频率限制
		if resp.StatusCode == 429 {
			return "", fmt.Errorf("请求过于频繁，请稍后再试")
		}
		return "", fmt.Errorf("HTTP状态码: %d", resp.StatusCode)
	}

	// 处理gzip压缩
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", fmt.Errorf("解压失败: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	return string(body), nil
}

// DownloadFile 下载文件
func (h *HTTPClient) DownloadFile(ctx context.Context, urlStr, filePath string, progress func(int64, int64)) error {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", getRandomUserAgent())
	if !strings.Contains(urlStr, "platform=android_tv_yst") && !strings.Contains(urlStr, "platform=android") {
		req.Header.Set("Referer", "https://www.bilibili.com/")
	}

	resp, err := h.Client.Do(req.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP状态码: %d", resp.StatusCode)
	}

	contentLength := resp.ContentLength
	if progress != nil {
		progress(0, contentLength)
	}

	// 创建文件
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	// 下载文件
	var downloaded int64
	buffer := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			_, writeErr := file.Write(buffer[:n])
			if writeErr != nil {
				return fmt.Errorf("写入文件失败: %w", writeErr)
			}
			downloaded += int64(n)

			if progress != nil {
				progress(downloaded, contentLength)
			}
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("下载失败: %w", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return nil
}

// getRandomUserAgent 生成随机User-Agent
func getRandomUserAgent() string {
	platforms := []string{
		"Windows NT 10.0; Win64; x64",
		"Macintosh; Intel Mac OS X 10_15_7",
		"X11; Linux x86_64",
	}

	browsers := []string{
		fmt.Sprintf("AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%.1f Safari/537.36", 80+rand.Float64()*30),
		fmt.Sprintf("Gecko/20100101 Firefox/%.1f", 80+rand.Float64()*30),
	}

	return fmt.Sprintf("Mozilla/5.0 (%s) %s", platforms[rand.Intn(len(platforms))], browsers[rand.Intn(len(browsers))])
}

// getTimestamp 获取时间戳
func getTimestamp() int64 {
	return time.Now().Unix()
}

// buildQueryStringHTTP 构建查询字符串
func buildQueryStringHTTP(params map[string]string) string {
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	return values.Encode()
}

// GetWBIKey 获取WBI密钥
func GetWBIKey(client *HTTPClient) (string, error) {
	api := "https://api.bilibili.com/x/web-interface/nav"

	resp, err := client.GetWebSource(api, "")
	if err != nil {
		return "", fmt.Errorf("获取WBI密钥失败: %w", err)
	}

	// 解析JSON响应
	var data map[string]interface{}
	err = json.Unmarshal([]byte(resp), &data)
	if err != nil {
		return "", fmt.Errorf("解析WBI响应失败: %w", err)
	}

	// 提取wbi_img
	wbiData, ok := data["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("WBI响应格式错误: data field missing")
	}

	wbiImg, ok := wbiData["wbi_img"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("WBI图片信息缺失: wbi_img field missing")
	}

	imgURL, ok := wbiImg["img_url"].(string)
	if !ok {
		return "", fmt.Errorf("WBI img_url缺失 or not string")
	}

	subURL, ok := wbiImg["sub_url"].(string)
	if !ok {
		return "", fmt.Errorf("WBI sub_url缺失 or not string")
	}

	// 提取文件名并生成密钥
	imgPart := util.RSubString(imgURL)
	subPart := util.RSubString(subURL)
	orig := imgPart + subPart

	wbiKey := util.GetMixinKey(orig)
	return wbiKey, nil
}
