package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/tekintian/go-bbdown/util"
)

// progressInfo 进度信息
type progressInfo struct {
	downloaded int64
	total      int64
}

// Download 下载视频
func Download(url string, config *Config) error {
	// 提取视频ID
	id, err := util.ExtractVideoID(url)
	if err != nil {
		return err
	}

	// 获取视频信息
	vinfo, err := fetchVideoInfo(id, config)
	if err != nil {
		return err
	}

	// 如果只需要信息，直接返回
	if config.OnlyShowInfo {
		fmt.Printf("视频信息：\n")
		fmt.Printf("标题：%s\n", vinfo.Title)
		fmt.Printf("UP主：%s\n", vinfo.Owner.Name)
		fmt.Printf("播放量：%d\n", vinfo.Stat.View)
		fmt.Printf("弹幕数：%d\n", vinfo.Stat.Danmaku)
		fmt.Printf("评论数：%d\n", vinfo.Stat.Reply)
		fmt.Printf("点赞数：%d\n", vinfo.Stat.Like)
		fmt.Printf("硬币数：%d\n", vinfo.Stat.Coin)
		fmt.Printf("收藏数：%d\n", vinfo.Stat.Favorite)
		return nil
	}

	// 选择分P
	var selectedPages []Page // 将[]*Page改为[]Page
	if config.SelectPage != "" {
		// 解析选择的分P
		selectedPages, err = parseSelectedPages(config.SelectPage, vinfo.Pages)
		if err != nil {
			return err
		}
	} else {
		selectedPages = vinfo.Pages
	}

	// 下载每个分P
	for _, page := range selectedPages {
		fmt.Printf("正在下载分P：%s\n", page.Part)

		// 提取音视频轨道
		parser := NewParser(config)
		aidStr := fmt.Sprintf("%d", vinfo.Aid)
		cidStr := fmt.Sprintf("%d", page.Cid)
		tracks, err := parser.ExtractTracks("", aidStr, aidStr, cidStr, "", config.UseTVApi, config.UseIntlApi, config.UseAppApi, "")
		if err != nil {
			return err
		}

		// 选择轨道
		var selectedVideoTrack *Track
		var selectedAudioTrack *Track
		var videoPath, audioPath string // 移到这里以扩大作用域

		// 处理视频轨道选择
		if !config.AudioOnly {
			selectedVideoTrack, err = selectVideoTrack(tracks, config)
			if err != nil {
				return err
			}
		}

		// 处理音频轨道选择
		if !config.VideoOnly {
			selectedAudioTrack, err = selectAudioTrack(tracks, config)
			if err != nil {
				return err
			}
		}

		// 下载轨道
		if selectedVideoTrack != nil {
			fmt.Printf("正在下载视频：%s\n", selectedVideoTrack.Description) // 将QualityDesc改为Description
			videoPath = fmt.Sprintf("%s_video.mp4", page.Part)
			err = downloadTrack(selectedVideoTrack, videoPath, config)
			if err != nil {
				return err
			}
		}

		if selectedAudioTrack != nil {
			fmt.Printf("正在下载音频：%s\n", selectedAudioTrack.Codec)
			audioPath = fmt.Sprintf("%s_audio.mp4", page.Part)
			err = downloadTrack(selectedAudioTrack, audioPath, config)
			if err != nil {
				return err
			}
		}

		// 混流
		if !config.SkipMux && selectedVideoTrack != nil && selectedAudioTrack != nil {
			fmt.Printf("正在混流...\n")
			// 生成输出文件名
			fileName := page.Part
			if fileName == "" {
				// 如果分P标题为空，使用视频标题
				fileName = vinfo.Title
			}

			// 如果有多个分P，在文件名中添加序号
			if len(vinfo.Pages) > 1 {
				fileName = fmt.Sprintf("%s_P%d", fileName, page.Index)
			}

			// 清理文件名中的非法字符
			fileName = strings.ReplaceAll(fileName, "/", "_")
			fileName = strings.ReplaceAll(fileName, "\\", "_")
			fileName = strings.ReplaceAll(fileName, ":", "_")
			fileName = strings.ReplaceAll(fileName, "*", "_")
			fileName = strings.ReplaceAll(fileName, "?", "_")
			fileName = strings.ReplaceAll(fileName, "\"", "_")
			fileName = strings.ReplaceAll(fileName, "<", "_")
			fileName = strings.ReplaceAll(fileName, ">", "_")
			fileName = strings.ReplaceAll(fileName, "|", "_")

			// 混流输出文件
			videoOutputPath := fmt.Sprintf("%s.mp4", fileName)
			err = muxTracks(selectedVideoTrack, selectedAudioTrack, videoPath, audioPath, videoOutputPath, config)
			if err != nil {
				return err
			}

			// 单独保存音频文件
			if selectedAudioTrack != nil {
				// 根据音频编码确定文件扩展名
				audioExt := ".m4a" // 默认扩展名
				if strings.Contains(strings.ToLower(selectedAudioTrack.Codec), "mp3") {
					audioExt = ".mp3"
				} else if strings.Contains(strings.ToLower(selectedAudioTrack.Codec), "aac") {
					audioExt = ".aac"
				} else if strings.Contains(strings.ToLower(selectedAudioTrack.Codec), "opus") {
					audioExt = ".opus"
				} else if strings.Contains(strings.ToLower(selectedAudioTrack.Codec), "flac") {
					audioExt = ".flac"
				}

				audioOutputPath := fmt.Sprintf("%s%s", fileName, audioExt)

				// 复制音频文件到最终位置
				err = util.CopyFile(audioPath, audioOutputPath)
				if err != nil {
					fmt.Printf("保存音频文件失败: %v\n", err)
				} else {
					fmt.Printf("音频已保存: %s\n", audioOutputPath)
				}
			}

			// 删除临时文件
			if !config.SimplyMux {
				os.Remove(videoPath)
				os.Remove(audioPath)
			}
		}

		fmt.Printf("分P下载完成：%s\n", page.Part)
	}

	return nil
}

// fetchVideoInfo 获取视频信息
func fetchVideoInfo(id string, config *Config) (*VInfo, error) {
	// 根据ID类型选择不同的API
	var api string
	var params map[string]string

	// 解析ID类型
	idType := ""
	if strings.HasPrefix(id, "BV") {
		idType = "bv"
		// 直接使用BV号，不需要转换
		params = map[string]string{
			"bvid": id,
		}
	} else if strings.HasPrefix(id, "av") {
		idType = "av"
		av := strings.TrimPrefix(id, "av")
		params = map[string]string{
			"aid": av,
		}
	} else if strings.HasPrefix(id, "ep") {
		idType = "ep"
		ep := strings.TrimPrefix(id, "ep")
		params = map[string]string{
			"ep_id": ep,
		}
	} else if strings.HasPrefix(id, "ss") {
		idType = "ss"
		ss := strings.TrimPrefix(id, "ss")
		params = map[string]string{
			"season_id": ss,
		}
	} else {
		return nil, fmt.Errorf("不支持的ID类型：%s", id)
	}

	// 创建HTTP客户端
	client := NewHTTPClient()

	// 构建API URL
	switch idType {
	case "bv", "av":
		prefix := fmt.Sprintf("https://%s/x/web-interface/view?", config.Host)

		// 添加WBI签名
		wbiKey, err := GetWBIKey(client)
		if err != nil {
			return nil, fmt.Errorf("获取WBI密钥失败: %w", err)
		}
		queryString := buildQueryStringHTTP(params)
		api = fmt.Sprintf("%s%s&w_rid=%s", prefix, queryString, util.WBISign(queryString, wbiKey))
	case "ep":
		api = fmt.Sprintf("https://%s/pgc/view/web/season?%s", config.EpHost, buildQueryStringHTTP(params))
	case "ss":
		api = fmt.Sprintf("https://%s/pgc/view/web/season?%s", config.EpHost, buildQueryStringHTTP(params))
	}

	if config.Debug {
		fmt.Printf("Debug: API URL: %s\n", api)
	}

	// 发送请求
	resp, err := client.GetWebSource(api, config.UserAgent)
	if err != nil {
		return nil, err
	}

	if config.Debug {
		fmt.Printf("Debug: Response: %s\n", resp)
	}

	// 解析响应 - API返回的数据在data字段中
	var response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    VInfo  `json:"data"`
	}

	err = parseJSON(resp, &response)
	if err != nil {
		return nil, err
	}

	if response.Code != 0 {
		return nil, fmt.Errorf("API错误: %s", response.Message)
	}

	return &response.Data, nil
}

// parseSelectedPages 解析选择的分P
func parseSelectedPages(selected string, pages []Page) ([]Page, error) { // 将[]*Page改为[]Page
	// 解析选择的分P，支持格式：1,2-5,7
	var selectedPages []Page // 将[]*Page改为[]Page

	// 分割选择的分P
	ranges := strings.Split(selected, ",")
	for _, r := range ranges {
		r = strings.TrimSpace(r)
		if strings.Contains(r, "-") {
			// 范围选择
			parts := strings.Split(r, "-")
			if len(parts) != 2 {
				return nil, fmt.Errorf("无效的分P选择：%s", r)
			}
			start, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, err
			}
			end, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, err
			}
			for i := start - 1; i < end; i++ {
				if i >= 0 && i < len(pages) {
					selectedPages = append(selectedPages, pages[i]) // 直接使用值类型
				}
			}
		} else {
			// 单个选择
			index, err := strconv.Atoi(r)
			if err != nil {
				return nil, err
			}
			if index-1 >= 0 && index-1 < len(pages) {
				selectedPages = append(selectedPages, pages[index-1]) // 直接使用值类型
			}
		}
	}

	return selectedPages, nil
}

// selectVideoTrack 选择视频轨道
func selectVideoTrack(tracks []*Track, config *Config) (*Track, error) {
	// 过滤视频轨道
	var videoTracks []*Track
	for _, track := range tracks {
		if track.FrameType == "video" {
			videoTracks = append(videoTracks, track)
		}
	}

	if len(videoTracks) == 0 {
		return nil, fmt.Errorf("没有找到视频轨道")
	}

	// 如果是交互式选择
	if config.Interactive {
		// 显示所有可选轨道
		fmt.Println("可用的视频轨道：")
		for i, track := range videoTracks {
			fmt.Printf("%d. %dx%d - %s - %s\n", i+1, track.Width, track.Height, track.Codec, formatSize(track.Size))
		}

		// 选择轨道
		fmt.Print("请选择要下载的轨道：")
		var choice int
		fmt.Scanln(&choice)
		if choice < 1 || choice > len(videoTracks) {
			return nil, fmt.Errorf("无效的选择")
		}
		return videoTracks[choice-1], nil
	}

	// 按照质量优先级选择
	if config.QualityPriority != nil && len(config.QualityPriority) > 0 {
		for _, quality := range config.QualityPriority {
			for _, track := range videoTracks {
				if strings.Contains(track.Description, quality) {
					return track, nil
				}
			}
		}
	}

	// 按照编码优先级选择
	if config.EncodingPriority != nil && len(config.EncodingPriority) > 0 {
		for _, encoding := range config.EncodingPriority {
			for _, track := range videoTracks {
				if strings.EqualFold(track.Codec, encoding) {
					return track, nil
				}
			}
		}
	}

	// 默认选择最高质量
	var bestTrack *Track
	for _, track := range videoTracks {
		if bestTrack == nil || track.Width > bestTrack.Width || track.Height > bestTrack.Height {
			bestTrack = track
		}
	}

	return bestTrack, nil
}

// selectAudioTrack 选择音频轨道
func selectAudioTrack(tracks []*Track, config *Config) (*Track, error) {
	// 过滤音频轨道
	var audioTracks []*Track
	for _, track := range tracks {
		if track.FrameType == "audio" {
			audioTracks = append(audioTracks, track)
		}
	}

	if len(audioTracks) == 0 {
		return nil, fmt.Errorf("没有找到音频轨道")
	}

	// 如果是交互式选择
	if config.Interactive {
		// 显示所有可选轨道
		fmt.Println("可用的音频轨道：")
		for i, track := range audioTracks {
			fmt.Printf("%d. %s - %s - %s\n", i+1, track.Codec, track.Description, formatSize(track.Size))
		}

		// 选择轨道
		fmt.Print("请选择要下载的轨道：")
		var choice int
		fmt.Scanln(&choice)
		if choice < 1 || choice > len(audioTracks) {
			return nil, fmt.Errorf("无效的选择")
		}
		return audioTracks[choice-1], nil
	}

	// 默认选择最高质量
	var bestTrack *Track
	for _, track := range audioTracks {
		if bestTrack == nil || track.Bandwidth > bestTrack.Bandwidth {
			bestTrack = track
		}
	}

	return bestTrack, nil
}

// downloadTrack 下载轨道
func downloadTrack(track *Track, path string, config *Config) error {
	// 创建目录
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	client := NewHTTPClient()

	// 使用aria2c
	if config.UseAria2c {
		return downloadWithAria2c(track.URL, path, config)
	}

	// 多线程下载
	if config.MultiThread {
		return multiThreadDownload(client, track.URL, path, config)
	}

	// 单线程下载
	return singleThreadDownload(client, track.URL, path, config)
}

// multiThreadDownload 多线程下载
func multiThreadDownload(client *HTTPClient, url, filePath string, config *Config) error {
	// 获取文件大小
	fileSize, err := getFileSize(client, url)
	if err != nil {
		return fmt.Errorf("获取文件大小失败: %w", err)
	}

	// 检查文件是否已存在且大小正确
	if info, err := os.Stat(filePath); err == nil && info.Size() == fileSize {
		fmt.Printf("文件已存在，跳过下载: %s\n", filePath)
		return nil
	}

	// 计算分段
	clips := getAllClips(url, fileSize)
	fmt.Printf("文件大小: %d bytes, 分段数量: %d\n", fileSize, len(clips))

	// 创建进度跟踪
	progress := make(chan progressInfo, len(clips))

	var totalDownloaded int64
	var wg sync.WaitGroup

	// 并发下载分段
	for _, clip := range clips {
		wg.Add(1)
		go func(c Clip) {
			defer wg.Done()
			err := downloadClip(client, url, filePath, c, progress)
			if err != nil {
				fmt.Printf("下载分段 %d 失败: %v\n", c.Index, err)
			}
		}(clip)
	}

	// 进度监控
	go func() {
		for p := range progress {
			atomic.AddInt64(&totalDownloaded, p.downloaded)
			percent := float64(totalDownloaded) / float64(fileSize) * 100
			fmt.Printf("\r下载进度: %.2f%% (%d/%d bytes)", percent, totalDownloaded, fileSize)
		}
	}()

	wg.Wait()
	close(progress)
	fmt.Println() // 换行

	// 合并文件
	if len(clips) > 1 {
		return mergeClips(filePath, clips)
	} else if len(clips) == 1 {
		// 单个分段，重命名临时文件
		tempPath := fmt.Sprintf("%s.%05d.tmp", filePath, clips[0].Index)
		err := os.Rename(tempPath, filePath)
		if err != nil {
			return fmt.Errorf("重命名文件失败: %w", err)
		}
	}

	return nil
}

// singleThreadDownload 单线程下载
func singleThreadDownload(client *HTTPClient, url, filePath string, config *Config) error {
	fmt.Printf("开始下载: %s\n", url)

	var lastProgress int64
	progressCallback := func(downloaded, total int64) {
		if total > 0 {
			percent := float64(downloaded) / float64(total) * 100
			fmt.Printf("\r下载进度: %.2f%% (%d/%d bytes)", percent, downloaded, total)
		}
		lastProgress = downloaded
	}

	err := client.DownloadFile(context.Background(), url, filePath, progressCallback)
	if err != nil && lastProgress > 0 {
		// 如果下载中断，尝试恢复
		fmt.Printf("\n下载中断，尝试恢复...\n")
		return resumeDownload(client, url, filePath, lastProgress, config)
	}

	if err != nil {
		return err
	}

	fmt.Println() // 换行
	return nil
}

// downloadWithAria2c 使用aria2c下载
func downloadWithAria2c(url, filePath string, config *Config) error {
	args := []string{
		"--summary-interval=1",
		"--show-console-readout=true",
		"--file-allocation=none",
		"--max-connection-per-server=16",
		"--split=16",
		"--min-split-size=1M",
	}

	if config.Aria2cArgs != "" {
		extraArgs := strings.Fields(config.Aria2cArgs)
		args = append(args, extraArgs...)
	}

	args = append(args, url, "-o", filePath)

	cmd := exec.Command(config.Aria2cPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// getFileSize 获取文件大小
func getFileSize(client *HTTPClient, url string) (int64, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("User-Agent", getRandomUserAgent())
	if !strings.Contains(url, "platform=android_tv_yst") && !strings.Contains(url, "platform=android") {
		req.Header.Set("Referer", "https://www.bilibili.com/")
	}

	resp, err := client.Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP状态码: %d", resp.StatusCode)
	}

	return resp.ContentLength, nil
}

// getAllClips 获取所有分段
func getAllClips(url string, fileSize int64) []Clip {
	var clips []Clip
	perSize := int64(20 * 1024 * 1024) // 20MB
	index := 0
	counter := int64(0)

	for fileSize > 0 {
		clip := Clip{
			Index: index,
			From:  counter,
			To:    counter + perSize,
		}

		if fileSize-perSize > 0 {
			fileSize -= perSize
			counter += perSize + 1
			index++
			clips = append(clips, clip)
		} else {
			clip.To = -1 // 表示到最后
			clips = append(clips, clip)
			break
		}
	}

	return clips
}

// downloadClip 下载单个分段
func downloadClip(client *HTTPClient, url, filePath string, clip Clip, progress chan<- progressInfo) error {
	tempPath := fmt.Sprintf("%s.%05d.tmp", filePath, clip.Index)

	// 检查临时文件是否已存在
	if info, err := os.Stat(tempPath); err == nil {
		// 如果文件大小符合预期，跳过下载
		expectedSize := clip.To - clip.From + 1
		if clip.To == -1 || info.Size() == expectedSize {
			progress <- progressInfo{downloaded: expectedSize, total: expectedSize}
			return nil
		}
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", getRandomUserAgent())
	if !strings.Contains(url, "platform=android_tv_yst") && !strings.Contains(url, "platform=android") {
		req.Header.Set("Referer", "https://www.bilibili.com/")
	}

	// 设置Range头
	if clip.To != -1 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", clip.From, clip.To))
	} else {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", clip.From))
	}

	resp, err := client.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("HTTP状态码: %d", resp.StatusCode)
	}

	// 创建临时文件
	file, err := os.Create(tempPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 下载数据
	buffer := make([]byte, 32*1024) // 32KB
	var downloaded int64

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			_, writeErr := file.Write(buffer[:n])
			if writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}

	progress <- progressInfo{downloaded: downloaded, total: downloaded}
	return nil
}

// mergeClips 合并分段文件
func mergeClips(filePath string, clips []Clip) error {
	outputFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	for _, clip := range clips {
		tempPath := fmt.Sprintf("%s.%05d.tmp", filePath, clip.Index)
		inputFile, err := os.Open(tempPath)
		if err != nil {
			return err
		}

		_, err = io.Copy(outputFile, inputFile)
		inputFile.Close()
		os.Remove(tempPath) // 删除临时文件

		if err != nil {
			return err
		}
	}

	return nil
}

// resumeDownload 恢复下载
func resumeDownload(client *HTTPClient, url, filePath string, downloaded int64, config *Config) error {
	// 获取文件总大小
	totalSize, err := getFileSize(client, url)
	if err != nil {
		return err
	}

	// 创建Range请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", getRandomUserAgent())
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-", downloaded))
	if !strings.Contains(url, "platform=android_tv_yst") && !strings.Contains(url, "platform=android") {
		req.Header.Set("Referer", "https://www.bilibili.com/")
	}

	resp, err := client.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("恢复下载失败，状态码: %d", resp.StatusCode)
	}

	// 追加到文件
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// 继续下载
	buffer := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			_, writeErr := file.Write(buffer[:n])
			if writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)

			percent := float64(downloaded) / float64(totalSize) * 100
			fmt.Printf("\r恢复进度: %.2f%% (%d/%d bytes)", percent, downloaded, totalSize)
		}

		if err != nil {
			if err == io.EOF {
				fmt.Println("\n恢复下载完成")
				break
			}
			return err
		}
	}

	return nil
}

// Clip 分段信息
type Clip struct {
	Index int
	From  int64
	To    int64
}

// muxTracks 混流
func muxTracks(videoTrack, audioTrack *Track, videoPath, audioPath, outputPath string, config *Config) error {
	// 使用FFmpeg混流
	cmd := []string{
		config.FFmpegPath,
		"-i", videoPath,
		"-i", audioPath,
		"-c:v", "copy",
		"-c:a", "copy",
		outputPath,
		"-y",
	}

	// 执行命令
	err := executeCommand(cmd)
	if err != nil {
		// 如果FFmpeg失败，尝试使用MP4Box
		if config.UseMP4Box {
			cmd := []string{
				config.Mp4boxPath,
				"-add", videoPath,
				"-add", audioPath,
				"-new", outputPath,
			}
			return executeCommand(cmd)
		}
		return err
	}

	return nil
}

// formatSize 格式化文件大小
func formatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.2fKB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.2fMB", float64(size)/(1024*1024))
	} else {
		return fmt.Sprintf("%.2fGB", float64(size)/(1024*1024*1024))
	}
}

// executeCommand 执行命令
func executeCommand(cmd []string) error {
	// 使用os/exec包执行命令
	command := exec.Command(cmd[0], cmd[1:]...)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("命令执行失败: %s, 输出: %s", err.Error(), string(output))
	}
	return nil
}

// parseJSON 解析JSON
func parseJSON(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}
