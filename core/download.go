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
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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

	// 检查是否是合集或媒体列表
	if strings.HasPrefix(id, "season:") {
		return downloadSeason(id, config)
	}
	if strings.HasPrefix(id, "medialist:") {
		return downloadMediaList(id, config)
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
		if selectedVideoTrack != nil && selectedAudioTrack != nil {
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
				fmt.Printf("混流失败: %v\n", err)
			} else {
				fmt.Printf("混流完成: %s\n", videoOutputPath)
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
				if videoPath != "" {
					if err := os.Remove(videoPath); err != nil {
						fmt.Printf("删除临时视频文件失败: %v\n", err)
					}
				}
				if audioPath != "" {
					if err := os.Remove(audioPath); err != nil {
						fmt.Printf("删除临时音频文件失败: %v\n", err)
					}
				}
			}
		} else if selectedAudioTrack != nil && selectedVideoTrack == nil {
			// 只有音频，直接重命名为正确的音频格式
			fmt.Printf("仅下载音频，重命名文件...\n")
			// 生成输出文件名
			fileName := page.Part
			if fileName == "" {
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
			
			// 重命名文件
			err := os.Rename(audioPath, audioOutputPath)
			if err != nil {
				fmt.Printf("重命名音频文件失败: %v\n", err)
			} else {
				fmt.Printf("音频已保存: %s\n", audioOutputPath)
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
	} else if strings.HasPrefix(id, "season:") {
		idType = "season"
		seasonData := strings.TrimPrefix(id, "season:")
		var seasonID string
		if strings.Contains(seasonData, ":") {
			// 新格式: season:用户ID:合集ID
			parts := strings.Split(seasonData, ":")
			if len(parts) >= 2 {
				seasonID = parts[1]
			} else {
				seasonID = seasonData
			}
		} else {
			// 旧格式: season:合集ID
			seasonID = seasonData
		}
		params = map[string]string{
			"season_id": seasonID,
		}
	} else if strings.HasPrefix(id, "medialist:") {
		idType = "medialist"
		medialistID := strings.TrimPrefix(id, "medialist:")
		params = map[string]string{
			"ml_id": medialistID,
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

// downloadSeason 下载合集
func downloadSeason(id string, config *Config) error {
	seasonData := strings.TrimPrefix(id, "season:")
	
	var userID, seasonID string
	if strings.Contains(seasonData, ":") {
		// 新格式: season:用户ID:合集ID
		parts := strings.Split(seasonData, ":")
		if len(parts) >= 2 {
			userID = parts[0]
			seasonID = parts[1]
		} else {
			seasonID = seasonData
		}
	} else {
		// 旧格式: season:合集ID
		seasonID = seasonData
	}

	// 获取合集信息
	seasonInfo, err := fetchSeasonInfo(userID, seasonID, config)
	if err != nil {
		return err
	}

	if config.OnlyShowInfo {
		fmt.Printf("合集信息：\n")
		fmt.Printf("标题：%s\n", seasonInfo.SeasonName)
		fmt.Printf("描述：%s\n", seasonInfo.Description)
		fmt.Printf("视频数量：%d\n", seasonInfo.TotalCount)
		for i, video := range seasonInfo.Videos {
			fmt.Printf("%d. %s (%s)\n", i+1, video.Title, formatDuration(video.Duration))
		}
		return nil
	}

	fmt.Printf("开始下载合集：%s\n", seasonInfo.SeasonName)
	fmt.Printf("包含 %d 个视频\n", seasonInfo.TotalCount)

	// 下载每个视频
	for i, video := range seasonInfo.Videos {
		fmt.Printf("\n[%d/%d] 下载视频：%s\n", i+1, len(seasonInfo.Videos), video.Title)

		// 如果缺少aid或cid，先获取视频信息
		if video.Aid == 0 || video.Cid == 0 {
			vinfo, err := fetchVideoInfo(video.Bvid, config)
			if err != nil {
				fmt.Printf("获取视频信息失败：%s，错误：%v\n", video.Title, err)
				continue
			}
			video.Aid = vinfo.Aid
			if len(vinfo.Pages) > 0 {
				video.Cid = vinfo.Pages[0].Cid
			}
		}

		// 转换为SeasonVideo类型
		seasonVideo := SeasonVideo{
			Aid:      video.Aid,
			Bvid:     video.Bvid,
			Cid:      video.Cid,
			Title:    video.Title,
			Duration: video.Duration,
			Cover:    video.Cover,
			Index:    video.Index,
			Part:     video.Part,
		}

		// 调用下载单个视频的函数
		err := downloadSingleVideoByInfo(seasonVideo, config)
		if err != nil {
			fmt.Printf("下载视频失败：%s，错误：%v\n", video.Title, err)
			continue
		}
	}

	fmt.Printf("\n合集下载完成：%s\n", seasonInfo.SeasonName)
	return nil
}

// downloadMediaList 下载媒体列表
func downloadMediaList(id string, config *Config) error {
	medialistID := strings.TrimPrefix(id, "medialist:")

	// 获取媒体列表信息
	medialistInfo, err := fetchMediaListInfo(medialistID, config)
	if err != nil {
		return err
	}

	if config.OnlyShowInfo {
		fmt.Printf("媒体列表信息：\n")
		fmt.Printf("标题：%s\n", medialistInfo.Title)
		fmt.Printf("描述：%s\n", medialistInfo.Description)
		fmt.Printf("视频数量：%d\n", medialistInfo.TotalCount)
		for i, video := range medialistInfo.Videos {
			fmt.Printf("%d. %s (%s)\n", i+1, video.Title, formatDuration(video.Duration))
		}
		return nil
	}

	fmt.Printf("开始下载媒体列表：%s\n", medialistInfo.Title)
	fmt.Printf("包含 %d 个视频\n", medialistInfo.TotalCount)

	// 下载每个视频
	for i, video := range medialistInfo.Videos {
		fmt.Printf("\n[%d/%d] 下载视频：%s\n", i+1, len(medialistInfo.Videos), video.Title)

		// 如果缺少aid或cid，先获取视频信息
		if video.Aid == 0 || video.Cid == 0 {
			vinfo, err := fetchVideoInfo(video.Bvid, config)
			if err != nil {
				fmt.Printf("获取视频信息失败：%s，错误：%v\n", video.Title, err)
				continue
			}
			video.Aid = vinfo.Aid
			if len(vinfo.Pages) > 0 {
				video.Cid = vinfo.Pages[0].Cid
			}
		}

		// 转换为SeasonVideo类型
		seasonVideo := SeasonVideo{
			Aid:      video.Aid,
			Bvid:     video.Bvid,
			Cid:      video.Cid,
			Title:    video.Title,
			Duration: video.Duration,
			Cover:    video.Cover,
			Index:    video.Index,
			Part:     video.Part,
		}

		// 调用下载单个视频的函数
		err := downloadSingleVideoByInfo(seasonVideo, config)
		if err != nil {
			fmt.Printf("下载视频失败：%s，错误：%v\n", video.Title, err)
			continue
		}
	}

	fmt.Printf("\n媒体列表下载完成：%s\n", medialistInfo.Title)
	return nil
}

// downloadSingleVideo 下载单个视频（重用原有逻辑）
func downloadSingleVideo(url string, config *Config) error {
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

	// 选择分P
	var selectedPages []Page
	if config.SelectPage != "" {
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
		var videoPath, audioPath string
		if selectedVideoTrack != nil {
			fmt.Printf("正在下载视频：%s\n", selectedVideoTrack.Description)
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
		if selectedVideoTrack != nil && selectedAudioTrack != nil {
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
				fmt.Printf("混流失败: %v\n", err)
			} else {
				fmt.Printf("混流完成: %s\n", videoOutputPath)
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
				if videoPath != "" {
					if err := os.Remove(videoPath); err != nil {
						fmt.Printf("删除临时视频文件失败: %v\n", err)
					}
				}
				if audioPath != "" {
					if err := os.Remove(audioPath); err != nil {
						fmt.Printf("删除临时音频文件失败: %v\n", err)
					}
				}
			}
		} else if selectedAudioTrack != nil && selectedVideoTrack == nil {
			// 只有音频，直接重命名为正确的音频格式
			fmt.Printf("仅下载音频，重命名文件...\n")
			// 生成输出文件名
			fileName := page.Part
			if fileName == "" {
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
			
			// 重命名文件
			err := os.Rename(audioPath, audioOutputPath)
			if err != nil {
				fmt.Printf("重命名音频文件失败: %v\n", err)
			} else {
				fmt.Printf("音频已保存: %s\n", audioOutputPath)
			}
		}

		fmt.Printf("分P下载完成：%s\n", page.Part)
	}

	return nil
}

// downloadSingleVideoByInfo 通过视频信息下载单个视频
func downloadSingleVideoByInfo(video SeasonVideo, config *Config) error {
	// 提取音视频轨道
	parser := NewParser(config)
	aidStr := fmt.Sprintf("%d", video.Aid)
	cidStr := fmt.Sprintf("%d", video.Cid)
	tracks, err := parser.ExtractTracks("", aidStr, aidStr, cidStr, "", config.UseTVApi, config.UseIntlApi, config.UseAppApi, "")
	if err != nil {
		return err
	}

	// 选择轨道
	var selectedVideoTrack *Track
	var selectedAudioTrack *Track

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
	var videoPath, audioPath string
	if selectedVideoTrack != nil {
		fmt.Printf("正在下载视频：%s\n", selectedVideoTrack.Description)
		videoPath = fmt.Sprintf("%s_video.mp4", video.Part)
		err = downloadTrack(selectedVideoTrack, videoPath, config)
		if err != nil {
			return err
		}
	}

	if selectedAudioTrack != nil {
		fmt.Printf("正在下载音频：%s\n", selectedAudioTrack.Codec)
		audioPath = fmt.Sprintf("%s_audio.mp4", video.Part)
		err = downloadTrack(selectedAudioTrack, audioPath, config)
		if err != nil {
			return err
		}
	}

	// 混流
	if selectedVideoTrack != nil && selectedAudioTrack != nil {
		fmt.Printf("正在混流...\n")
		// 生成输出文件名
		fileName := video.Part
		if fileName == "" {
			// 如果分P标题为空，使用视频标题
			fileName = video.Title
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
			fmt.Printf("混流失败: %v\n", err)
		} else {
			fmt.Printf("混流完成: %s\n", videoOutputPath)
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
			if videoPath != "" {
				if err := os.Remove(videoPath); err != nil {
					fmt.Printf("删除临时视频文件失败: %v\n", err)
				}
			}
			if audioPath != "" {
				if err := os.Remove(audioPath); err != nil {
					fmt.Printf("删除临时音频文件失败: %v\n", err)
				}
			}
		}
	} else if selectedAudioTrack != nil && selectedVideoTrack == nil {
		// 只有音频，直接重命名为正确的音频格式
		fmt.Printf("仅下载音频，重命名文件...\n")
		// 生成输出文件名
		fileName := video.Part
		if fileName == "" {
			fileName = video.Title
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
		
		// 重命名文件
		err := os.Rename(audioPath, audioOutputPath)
		if err != nil {
			fmt.Printf("重命名音频文件失败: %v\n", err)
		} else {
			fmt.Printf("音频已保存: %s\n", audioOutputPath)
		}
	}

	return nil
}

// fetchSeasonInfo 获取合集信息
func fetchSeasonInfo(userID, seasonID string, config *Config) (*SeasonInfo, error) {
	client := NewHTTPClient()

	// 先尝试网页解析
	var webURL string
	if userID != "" {
		webURL = fmt.Sprintf("https://space.bilibili.com/%s/lists/%s?type=season", userID, seasonID)
	} else {
		// 如果没有用户ID，使用默认值（向后兼容）
		webURL = fmt.Sprintf("https://space.bilibili.com/89320896/lists/%s?type=season", seasonID)
	}
	
	webResp, webErr := client.GetWebSource(webURL, config.UserAgent)
	if webErr == nil {
		// 如果网页解析成功，尝试解析
		if seasonInfo, err := parseSeasonFromWeb(webResp, seasonID); err == nil && seasonInfo != nil {
			return seasonInfo, nil
		}
	}

	// 网页解析失败，再尝试API
	// 先尝试合集API
	apiEndpoints := []string{
		fmt.Sprintf("https://api.bilibili.com/x/space/season/video_list?season_id=%s&ps=30&jsonp=jsonp", seasonID),
		fmt.Sprintf("https://api.bilibili.com/x/season/archives?season_id=%s", seasonID),
		fmt.Sprintf("https://api.bilibili.com/x/space/arc/search?mid=%s&ps=30&jsonp=jsonp", userID),
	}
	
	var resp string
	var err error
	
	maxRetries := 1 // 最大重试次数
	for retry := 0; retry < maxRetries; retry++ {
		for i, api := range apiEndpoints {
			// 添加延迟避免频率限制
			if i > 0 || retry > 0 {
				time.Sleep(2 * time.Second)
			}
			
			resp, err = client.GetWebSource(api, config.UserAgent)
			if err == nil {
				// 检查是否是频率限制错误
				if strings.Contains(resp, "请求过于频繁") {
					time.Sleep(5 * time.Second)
					continue
				}
				
				// 检查响应是否为错误页面
				if !strings.Contains(resp, "出错啦") && !strings.Contains(resp, "<title>出错") {
					// 尝试不同的解析方式
					parsers := []func(string) (*SeasonInfo, error){
						parseSeasonFromSpaceAPI,
						parseSeasonFromMedialistAPI,
						parseSeasonFromSpaceSeasonAPI,
					}

					for _, parser := range parsers {
						if seasonInfo, err := parser(resp); err == nil && seasonInfo != nil {
							return seasonInfo, nil
						}
					}
				}
			} else {
				// 如果是频率限制错误，等待更长时间
				if strings.Contains(err.Error(), "过于频繁") {
					time.Sleep(10 * time.Second)
					// 重新开始当前重试循环
					i--
					continue
				}
			}
		}
		
		// 如果已经尝试了所有API但都失败，且还有重试次数，则等待后重试
		if retry < maxRetries-1 {
			time.Sleep(15 * time.Second)
		}
	}
	
	// 如果合集API都失败，尝试作为收藏夹处理
	favAPI := fmt.Sprintf("https://api.bilibili.com/x/v3/fav/resource/list?media_id=%s&ps=30", seasonID)
	resp, err = client.GetWebSource(favAPI, config.UserAgent)
	if err == nil {
		// 如果收藏夹API成功，使用收藏夹解析器
		if favInfo, err := parseFavoriteAPI(resp, seasonID); err == nil && favInfo != nil && len(favInfo.Videos) > 0 {
			// 转换为SeasonInfo格式
			return &SeasonInfo{
				SeasonID:    seasonID,
				SeasonName:  favInfo.Title,
				Description: favInfo.Description,
				TotalCount:  favInfo.TotalCount,
				Videos:      convertFavVideosToSeasonVideos(favInfo.Videos),
			}, nil
		}
	}
	
	return nil, fmt.Errorf("所有解析方式都失败。可能原因：1）合集/收藏夹不存在或已被删除 2）合集/收藏夹为私有 3）用户ID或ID错误")
}

// parseSeasonFromSpaceAPI 从空间API解析合集信息
func parseSeasonFromSpaceAPI(resp string) (*SeasonInfo, error) {
	var response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			SeasonID    string `json:"season_id"`
			SeasonName  string `json:"season_name"`
			Description string `json:"description"`
			TotalCount  int    `json:"total"`
			Archives    []struct {
				Aid      int64  `json:"aid"`
				Bvid     string `json:"bvid"`
				Cid      int64  `json:"cid"`
				Title    string `json:"title"`
				Duration int    `json:"duration"`
				Cover    string `json:"cover"`
				Index    int    `json:"index"`
				Part     string `json:"part"`
			} `json:"archives"`
		} `json:"data"`
	}

	err := parseJSON(resp, &response)
	if err != nil || response.Code != 0 {
		return nil, fmt.Errorf("API响应失败或解析错误")
	}

	// 转换为SeasonInfo结构
	seasonInfo := &SeasonInfo{
		SeasonID:    response.Data.SeasonID,
		SeasonName:  response.Data.SeasonName,
		Description: response.Data.Description,
		TotalCount:  response.Data.TotalCount,
		Videos:      make([]SeasonVideo, len(response.Data.Archives)),
	}

	for i, archive := range response.Data.Archives {
		seasonInfo.Videos[i] = SeasonVideo{
			Aid:      archive.Aid,
			Bvid:     archive.Bvid,
			Cid:      archive.Cid,
			Title:    archive.Title,
			Duration: archive.Duration,
			Cover:    archive.Cover,
			Index:    archive.Index,
			Part:     archive.Part,
		}
	}

	return seasonInfo, nil
}

// parseSeasonFromMedialistAPI 从medialist API解析合集信息
func parseSeasonFromMedialistAPI(resp string) (*SeasonInfo, error) {
	var medialistResponse struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			Description string `json:"description"`
			MediaCount  int    `json:"media_count"`
			MediaList   []struct {
				Aid      int64  `json:"aid"`
				Bvid     string `json:"bvid"`
				Cid      int64  `json:"cid"`
				Title    string `json:"title"`
				Duration int    `json:"duration"`
				Cover    string `json:"cover"`
				Index    int    `json:"index"`
				Part     string `json:"part"`
			} `json:"medias"`
		} `json:"data"`
	}

	err := parseJSON(resp, &medialistResponse)
	if err != nil || medialistResponse.Code != 0 || len(medialistResponse.Data.MediaList) == 0 {
		return nil, fmt.Errorf("medialist API解析失败")
	}

	// 转换为SeasonInfo结构
	seasonInfo := &SeasonInfo{
		SeasonID:    medialistResponse.Data.ID,
		SeasonName:  medialistResponse.Data.Title,
		Description: medialistResponse.Data.Description,
		TotalCount:  medialistResponse.Data.MediaCount,
		Videos:      make([]SeasonVideo, len(medialistResponse.Data.MediaList)),
	}

	for i, media := range medialistResponse.Data.MediaList {
		seasonInfo.Videos[i] = SeasonVideo{
			Aid:      media.Aid,
			Bvid:     media.Bvid,
			Cid:      media.Cid,
			Title:    media.Title,
			Duration: media.Duration,
			Cover:    media.Cover,
			Index:    i + 1,
			Part:     media.Part,
		}
	}

	return seasonInfo, nil
}

// parseSeasonFromSpaceSeasonAPI 从space season API解析合集信息
func parseSeasonFromSpaceSeasonAPI(resp string) (*SeasonInfo, error) {
	var spaceSeasonResponse struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			SeasonID    string `json:"season_id"`
			SeasonName  string `json:"season_name"`
			Description string `json:"description"`
			TotalCount  int    `json:"total"`
			Archives    []struct {
				Aid      int64  `json:"aid"`
				Bvid     string `json:"bvid"`
				Cid      int64  `json:"cid"`
				Title    string `json:"title"`
				Duration int    `json:"duration"`
				Cover    string `json:"cover"`
				Index    int    `json:"index"`
				Part     string `json:"part"`
			} `json:"archives"`
		} `json:"data"`
	}

	err := parseJSON(resp, &spaceSeasonResponse)
	if err != nil || spaceSeasonResponse.Code != 0 || len(spaceSeasonResponse.Data.Archives) == 0 {
		return nil, fmt.Errorf("space season API解析失败")
	}

	// 转换为SeasonInfo结构
	seasonInfo := &SeasonInfo{
		SeasonID:    spaceSeasonResponse.Data.SeasonID,
		SeasonName:  spaceSeasonResponse.Data.SeasonName,
		Description: spaceSeasonResponse.Data.Description,
		TotalCount:  spaceSeasonResponse.Data.TotalCount,
		Videos:      make([]SeasonVideo, len(spaceSeasonResponse.Data.Archives)),
	}

	for i, archive := range spaceSeasonResponse.Data.Archives {
		seasonInfo.Videos[i] = SeasonVideo{
			Aid:      archive.Aid,
			Bvid:     archive.Bvid,
			Cid:      archive.Cid,
			Title:    archive.Title,
			Duration: archive.Duration,
			Cover:    archive.Cover,
			Index:    archive.Index,
			Part:     archive.Part,
		}
	}

	return seasonInfo, nil
}

// parseSeasonFromWeb 从网页解析合集信息
func parseSeasonFromWeb(html, seasonID string) (*SeasonInfo, error) {
	
	
	// 使用正则表达式提取合集信息
	// 查找初始化数据的JSON（新格式）
	var jsonRegex *regexp.Regexp
	var matches [][]string
	
	// 尝试新格式的初始化数据
	jsonRegex = regexp.MustCompile(`<script>window\.__INITIAL_STATE__\s*=\s*({.*?});</script>`)
	matches = jsonRegex.FindAllStringSubmatch(html, -1)
	
	if len(matches) == 0 || len(matches[0]) < 2 {
		// 尝试旧格式
		jsonRegex = regexp.MustCompile(`window\.__INITIAL_STATE__\s*=\s*({.*?});`)
		oldMatches := jsonRegex.FindStringSubmatch(html)
		if len(oldMatches) >= 2 {
			matches = [][]string{oldMatches}
		}
	}
	
	if len(matches) == 0 || len(matches[0]) < 2 {
		// 尝试直接从网页中提取基本信息
		return extractBasicSeasonInfo(html, seasonID)
	}

	jsonStr := matches[0][1]

	// 解析JSON
	var initialState map[string]interface{}
	err := parseJSON(jsonStr, &initialState)
	if err != nil {
		return nil, fmt.Errorf("解析初始化数据失败: %w", err)
	}

	// 尝试从不同的路径获取合集信息
	var seasonData map[string]interface{}

	// 尝试路径1
	if space, ok := initialState["space"].(map[string]interface{}); ok {
		if seasons, ok := space["seasons"].(map[string]interface{}); ok {
			if currentSeason, ok := seasons[seasonID].(map[string]interface{}); ok {
				seasonData = currentSeason
			}
		}
	}

	// 尝试路径2
	if seasonData == nil {
		if seasons, ok := initialState["seasons"].(map[string]interface{}); ok {
			if currentSeason, ok := seasons[seasonID].(map[string]interface{}); ok {
				seasonData = currentSeason
			}
		}
	}

	if seasonData == nil {
		// 尝试从网页中提取基本信息
		return extractBasicSeasonInfo(html, seasonID)
	}

	// 提取基本信息
	seasonName := getStringFromMap(seasonData, "name")
	description := getStringFromMap(seasonData, "description")

	// 提取视频列表
	var videos []SeasonVideo
	if archives, ok := seasonData["archives"].([]interface{}); ok {
		for _, archive := range archives {
			if archiveMap, ok := archive.(map[string]interface{}); ok {
				video := SeasonVideo{
					Aid:      getInt64FromMap(archiveMap, "aid"),
					Bvid:     getStringFromMap(archiveMap, "bvid"),
					Cid:      getInt64FromMap(archiveMap, "cid"),
					Title:    getStringFromMap(archiveMap, "title"),
					Duration: getIntFromMap(archiveMap, "duration"),
					Cover:    getStringFromMap(archiveMap, "cover"),
					Index:    getIntFromMap(archiveMap, "index"),
					Part:     getStringFromMap(archiveMap, "part"),
				}
				videos = append(videos, video)
			}
		}
	}

	return &SeasonInfo{
		SeasonID:    seasonID,
		SeasonName:  seasonName,
		Description: description,
		TotalCount:  len(videos),
		Videos:      videos,
	}, nil
}

// extractBasicSeasonInfo 从网页中提取基本合集信息（备用方案）
func extractBasicSeasonInfo(html, seasonID string) (*SeasonInfo, error) {
	// 检查是否是验证码页面
	if strings.Contains(html, "验证码_哔哩哔哩") || strings.Contains(html, "risk-captcha") {
		return nil, fmt.Errorf("遇到验证码页面，无法访问合集内容")
	}
	
	// 检查是否是错误页面
	if strings.Contains(html, "出错啦") || strings.Contains(html, "<title>出错") {
		return nil, fmt.Errorf("页面不存在或访问出错")
	}
	
	// 提取合集标题
	titleRegex := regexp.MustCompile(`<h1[^>]*class="[^"]*title[^"]*"[^>]*>([^<]+)</h1>`)
	titleMatches := titleRegex.FindStringSubmatch(html)
	seasonName := "未知合集"
	if len(titleMatches) > 1 {
		seasonName = strings.TrimSpace(titleMatches[1])
	}
	
	// 如果没有找到标题，尝试其他方式
	if seasonName == "未知合集" {
		titleRegex2 := regexp.MustCompile(`"title":"([^"]+)"`)
		titleMatches2 := titleRegex2.FindStringSubmatch(html)
		if len(titleMatches2) > 1 {
			seasonName = strings.TrimSpace(titleMatches2[1])
		}
	}
	
	// 如果仍然是验证码页面标题，直接返回错误
	if seasonName == "验证码_哔哩哔哩" {
		return nil, fmt.Errorf("遇到验证码页面，无法访问合集内容")
	}
	
	// 提取视频列表（新格式尝试）
	videoRegex := regexp.MustCompile(`<a[^>]*href="([^"]*video/([^"]*))"[^>]*>.*?<span[^>]*class="[^"]*title[^"]*"[^>]*>([^<]+)</span>`)
	videoMatches := videoRegex.FindAllStringSubmatch(html, -1)
	
	// 如果新格式没有找到，尝试更简单的模式
	if len(videoMatches) == 0 {
		videoRegex = regexp.MustCompile(`<a[^>]*href="/video/([^"]+)"[^>]*>([^<]+)</a>`)
		videoMatches = videoRegex.FindAllStringSubmatch(html, -1)
	}

	var videos []SeasonVideo
	for i, match := range videoMatches {
		if len(match) >= 4 {
			bvid := match[2]
			title := strings.TrimSpace(match[3])

			video := SeasonVideo{
				Aid:      0, // 需要通过bvid获取
				Bvid:     bvid,
				Cid:      0, // 需要后续获取
				Title:    title,
				Duration: 0, // 需要后续获取
				Cover:    "",
				Index:    i + 1,
				Part:     title,
			}
			videos = append(videos, video)
		}
	}

	if len(videos) == 0 {
		return nil, fmt.Errorf("无法从网页中提取视频列表")
	}

	return &SeasonInfo{
		SeasonID:    seasonID,
		SeasonName:  seasonName,
		Description: "",
		TotalCount:  len(videos),
		Videos:      videos,
	}, nil
}

// getStringFromMap 从map中安全获取字符串值
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// getIntFromMap 从map中安全获取int值
func getIntFromMap(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		if f, ok := val.(float64); ok {
			return int(f)
		}
		if i, ok := val.(int); ok {
			return i
		}
	}
	return 0
}

// getInt64FromMap 从map中安全获取int64值
func getInt64FromMap(m map[string]interface{}, key string) int64 {
	if val, ok := m[key]; ok {
		if f, ok := val.(float64); ok {
			return int64(f)
		}
		if i, ok := val.(int); ok {
			return int64(i)
		}
		if i64, ok := val.(int64); ok {
			return i64
		}
	}
	return 0
}

// parseFavoriteAPI 解析收藏夹API响应
func parseFavoriteAPI(resp string, favID string) (*FavoriteInfo, error) {
	var response struct {
		Code    int `json:"code"`
		Message string `json:"message"`
		Data    *struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			Description string `json:"description"`
			MediaCount  int    `json:"media_count"`
			Medias      []struct {
				Aid      int64  `json:"aid"`
				Bvid     string `json:"bvid"`
				Cid      int64  `json:"cid"`
				Title    string `json:"title"`
				Duration int    `json:"duration"`
				Cover    string `json:"cover"`
				Upper    struct {
					Name string `json:"name"`
				} `json:"upper"`
			} `json:"medias"`
		} `json:"data"`
	}

	err := parseJSON(resp, &response)
	if err != nil || response.Code != 0 {
		return nil, fmt.Errorf("收藏夹API解析失败")
	}

	if response.Data == nil || len(response.Data.Medias) == 0 {
		return nil, fmt.Errorf("收藏夹为空或不存在")
	}

	// 转换为FavoriteInfo结构
	favInfo := &FavoriteInfo{
		ID:          response.Data.ID,
		Title:       response.Data.Title,
		Description: response.Data.Description,
		TotalCount:  response.Data.MediaCount,
		Videos:      make([]FavoriteVideo, len(response.Data.Medias)),
	}

	for i, media := range response.Data.Medias {
		favInfo.Videos[i] = FavoriteVideo{
			Aid:      media.Aid,
			Bvid:     media.Bvid,
			Cid:      media.Cid,
			Title:    media.Title,
			Duration: media.Duration,
			Cover:    media.Cover,
			Part:     media.Title, // 收藏夹中没有part字段，使用title
		}
	}

	return favInfo, nil
}

// convertFavVideosToSeasonVideos 将收藏夹视频转换为合集视频格式
func convertFavVideosToSeasonVideos(favVideos []FavoriteVideo) []SeasonVideo {
	videos := make([]SeasonVideo, len(favVideos))
	for i, fav := range favVideos {
		videos[i] = SeasonVideo{
			Aid:      fav.Aid,
			Bvid:     fav.Bvid,
			Cid:      fav.Cid,
			Title:    fav.Title,
			Duration: fav.Duration,
			Cover:    fav.Cover,
			Index:    i + 1,
			Part:     fav.Part,
		}
	}
	return videos
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// fetchMediaListInfo 获取媒体列表信息
func fetchMediaListInfo(medialistID string, config *Config) (*MediaListInfo, error) {
	client := NewHTTPClient()

	// 获取媒体列表信息API
	api := fmt.Sprintf("https://api.bilibili.com/x/v2/medialist/info?ml_id=%s", medialistID)

	resp, err := client.GetWebSource(api, config.UserAgent)
	if err != nil {
		return nil, err
	}

	// 解析响应
	var response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			Description string `json:"description"`
			MediaCount  int    `json:"media_count"`
			MediaList   []struct {
				Aid      int64  `json:"aid"`
				Bvid     string `json:"bvid"`
				Cid      int64  `json:"cid"`
				Title    string `json:"title"`
				Duration int    `json:"duration"`
				Cover    string `json:"cover"`
				Index    int    `json:"index"`
				Part     string `json:"part"`
			} `json:"medias"`
		} `json:"data"`
	}

	err = parseJSON(resp, &response)
	if err != nil {
		return nil, err
	}

	if response.Code != 0 {
		return nil, fmt.Errorf("API错误: %s", response.Message)
	}

	// 转换为MediaListInfo结构
	medialistInfo := &MediaListInfo{
		ID:          response.Data.ID,
		Title:       response.Data.Title,
		Description: response.Data.Description,
		TotalCount:  response.Data.MediaCount,
		Videos:      make([]MediaVideo, len(response.Data.MediaList)),
	}

	for i, media := range response.Data.MediaList {
		medialistInfo.Videos[i] = MediaVideo{
			Aid:      media.Aid,
			Bvid:     media.Bvid,
			Cid:      media.Cid,
			Title:    media.Title,
			Duration: media.Duration,
			Cover:    media.Cover,
			Index:    media.Index,
			Part:     media.Part,
		}
	}

	return medialistInfo, nil
}

// formatDuration 格式化时长
func formatDuration(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60

	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}
