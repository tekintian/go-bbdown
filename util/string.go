package util

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// BVConverter BV号转换器
type BVConverter struct {
	Alphabet    string
	RevAlphabet map[byte]int64
	XorCode     int64
	MaskCode    int64
	MaxAid      int64
	MinAid      int64
	Base        int64
	BvLen       int
}

// 正则表达式
var (
	bvRegex         = regexp.MustCompile(`BV([a-zA-Z0-9]+)`)
	avRegex         = regexp.MustCompile(`av(\d+)`)
	epRegex         = regexp.MustCompile(`/ep(\d+)`)
	ssRegex         = regexp.MustCompile(`/ss(\d+)`)
	cheeseEpRegex   = regexp.MustCompile(`/cheese/play/ep(\d+)`)
	cheeseSsRegex   = regexp.MustCompile(`/cheese/play/ss(\d+)`)
	collectionRegex = regexp.MustCompile(`business_id=(\d+)`)
	seriesRegex     = regexp.MustCompile(`business_id=(\d+)`)
	seasonRegex     = regexp.MustCompile(`space\.bilibili\.com/(\d+)/lists/(\d+)`)
	medialistRegex  = regexp.MustCompile(`medialist/detail/ml(\d+)`)
)

// NewBVConverter 创建BV转换器
func NewBVConverter() *BVConverter {
	alphabet := "FcwAPNKTMug3GV5Lj7EJnHpWsx4tb8haYeviqBz6rkCy12mUSDQX9RdoZf"
	revAlphabet := make(map[byte]int64)
	for i := 0; i < len(alphabet); i++ {
		revAlphabet[alphabet[i]] = int64(i)
	}

	// 修复类型不匹配问题，将maskCode显式声明为int64
	maskCode := int64((1 << 51) - 1)

	return &BVConverter{
		Alphabet:    alphabet,
		RevAlphabet: revAlphabet,
		XorCode:     23442827791579,
		MaskCode:    maskCode,
		MaxAid:      maskCode + 1,
		MinAid:      1,
		Base:        58,
		BvLen:       10, // BV号主体部分为10个字符，总长度为12个字符（包括"BV"前缀）
	}
}

// BVToAV BV号转AV号
func (c *BVConverter) BVToAV(bv string) (string, error) {
	if !strings.HasPrefix(bv, "BV") {
		return "", fmt.Errorf("无效的BV号格式")
	}

	// 提取BV号主体部分（去掉"BV"前缀）
	bvidStr := bv[2:]

	if len(bvidStr) < c.BvLen {
		return "", fmt.Errorf("无效的BV号格式")
	}

	// 转换为字节数组并进行位置交换
	bvid := []byte(bvidStr)

	// 位置交换：(0,6) 和 (1,4)
	bvid[0], bvid[6] = bvid[6], bvid[0]
	bvid[1], bvid[4] = bvid[4], bvid[1]

	// 解码
	var avid int64
	for _, b := range bvid {
		index, ok := c.RevAlphabet[b]
		if !ok {
			return "", fmt.Errorf("无效的BV号字符: %c", b)
		}
		avid = avid*c.Base + index
	}

	// 应用掩码和XOR
	avid = (avid & c.MaskCode) ^ c.XorCode

	return strconv.FormatInt(avid, 10), nil
}

// AVToBV AV号转BV号
func (c *BVConverter) AVToBV(av string) (string, error) {
	avid, err := strconv.ParseInt(av, 10, 64)
	if err != nil {
		return "", fmt.Errorf("无效的AV号: %v", err)
	}

	if avid < c.MinAid {
		return "", fmt.Errorf("AV号 %d 小于最小值 %d", avid, c.MinAid)
	}
	if avid >= c.MaxAid {
		return "", fmt.Errorf("AV号 %d 大于等于最大值 %d", avid, c.MaxAid)
	}

	// 应用XOR和掩码
	tmp := (c.MaxAid | avid) ^ c.XorCode

	// 初始化结果数组，默认填充第一个字符
	bvid := make([]byte, c.BvLen)
	for i := range bvid {
		bvid[i] = c.Alphabet[0]
	}

	// 编码
	for i := c.BvLen - 1; tmp != 0; i-- {
		bvid[i] = c.Alphabet[tmp%c.Base]
		tmp /= c.Base
	}

	// 位置交换：(0,6) 和 (1,4)
	bvid[0], bvid[6] = bvid[6], bvid[0]
	bvid[1], bvid[4] = bvid[4], bvid[1]

	// 组合结果
	return "BV" + string(bvid), nil
}

// IsValidID 检查ID是否有效
func IsValidID(id string) bool {
	if strings.HasPrefix(id, "BV") {
		return len(id) == 12 // 现代BV号总长度为12个字符（包括"BV"前缀）
	}
	if strings.HasPrefix(id, "av") {
		num := strings.TrimPrefix(id, "av")
		_, err := strconv.ParseInt(num, 10, 64)
		return err == nil
	}
	if strings.HasPrefix(id, "ep") || strings.HasPrefix(id, "ss") {
		num := strings.TrimPrefix(id, "ep")
		num = strings.TrimPrefix(num, "ss")
		_, err := strconv.ParseInt(num, 10, 64)
		return err == nil
	}
	return false
}

// ExtractFromURL 从URL中提取ID
func ExtractFromURL(url string) (string, string, error) {
	// BV号
	if bvRegex.MatchString(url) {
		matches := bvRegex.FindStringSubmatch(url)
		if len(matches) > 1 {
			return "bv", matches[1], nil
		}
	}

	// AV号
	if avRegex.MatchString(url) {
		matches := avRegex.FindStringSubmatch(url)
		if len(matches) > 1 {
			return "av", matches[1], nil
		}
	}

	// EP号
	if epRegex.MatchString(url) {
		matches := epRegex.FindStringSubmatch(url)
		if len(matches) > 1 {
			return "ep", matches[1], nil
		}
	}

	// SS号
	if ssRegex.MatchString(url) {
		matches := ssRegex.FindStringSubmatch(url)
		if len(matches) > 1 {
			return "ss", matches[1], nil
		}
	}

	// 课程EP号
	if cheeseEpRegex.MatchString(url) {
		matches := cheeseEpRegex.FindStringSubmatch(url)
		if len(matches) > 1 {
			return "cheese_ep", matches[1], nil
		}
	}

	// 课程SS号
	if cheeseSsRegex.MatchString(url) {
		matches := cheeseSsRegex.FindStringSubmatch(url)
		if len(matches) > 1 {
			return "cheese_ss", matches[1], nil
		}
	}

	// 合集
	if collectionRegex.MatchString(url) {
		matches := collectionRegex.FindStringSubmatch(url)
		if len(matches) > 1 {
			return "collection", matches[1], nil
		}
	}

	// 系列
	if seriesRegex.MatchString(url) {
		matches := seriesRegex.FindStringSubmatch(url)
		if len(matches) > 1 {
			return "series", matches[1], nil
		}
	}

	// 合集列表
	if seasonRegex.MatchString(url) {
		matches := seasonRegex.FindStringSubmatch(url)
		if len(matches) > 2 {
			// 返回格式 "season:用户ID:合集ID"
			return "season", matches[1] + ":" + matches[2], nil
		}
		// 兼容旧格式，只包含合集ID
		if len(matches) > 1 {
			return "season", matches[1], nil
		}
	}

	// 媒体列表
	if medialistRegex.MatchString(url) {
		matches := medialistRegex.FindStringSubmatch(url)
		if len(matches) > 1 {
			return "medialist", matches[1], nil
		}
	}

	// 直接匹配
	if strings.HasPrefix(url, "BV") {
		return "bv", url, nil
	}
	if strings.HasPrefix(url, "av") {
		return "av", strings.TrimPrefix(url, "av"), nil
	}
	if strings.HasPrefix(url, "ep") {
		return "ep", strings.TrimPrefix(url, "ep"), nil
	}
	if strings.HasPrefix(url, "ss") {
		return "ss", strings.TrimPrefix(url, "ss"), nil
	}

	return "", "", fmt.Errorf("无法识别的URL格式")
}

// GetQueryString 获取URL查询参数
func GetQueryString(url, key string) string {
	re := regexp.MustCompile(`[?&]` + regexp.QuoteMeta(key) + `=([^&#]*)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// MD5Hash 计算MD5哈希
func MD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

// Sign 生成签名
func Sign(params, appKey string, appSecret string) string {
	text := params + appKey + appSecret
	return MD5Hash(text)
}

// WBISign WBI签名
func WBISign(api string, wbiKey string) string {
	text := api + wbiKey
	return MD5Hash(text)
}

// GetMixinKey 获取WBI混合密钥
func GetMixinKey(orig string) string {
	mixinKeyEncTab := []int{
		46, 47, 18, 2, 53, 8, 23, 32, 15, 50, 10, 31, 58, 3, 45, 35,
		27, 43, 5, 49, 33, 9, 42, 19, 29, 28, 14, 39, 12, 38, 41, 13,
	}

	var tmp strings.Builder
	for _, index := range mixinKeyEncTab {
		if index < len(orig) {
			tmp.WriteByte(orig[index])
		}
	}
	return tmp.String()
}

// RSubString 提取文件名部分
func RSubString(sub string) string {
	lastSlash := strings.LastIndex(sub, "/")
	if lastSlash == -1 {
		return sub
	}
	sub = sub[lastSlash+1:]

	lastDot := strings.LastIndex(sub, ".")
	if lastDot == -1 {
		return sub
	}
	return sub[:lastDot]
}

// GetWBIKey 获取WBI密钥
func GetWBIKey(client interface{}) (string, error) {
	// 这里需要传入HTTP客户端来获取WBI密钥
	// 暂时返回空字符串，稍后在core包中实现
	return "", nil
}

// FormatTimestamp 格式化时间戳
func FormatTimestamp(timestamp int64, format string) string {
	if timestamp == 0 {
		return ""
	}
	if format == "unix" {
		return fmt.Sprintf("%d", timestamp)
	}
	// 这里应该使用time包进行格式化，为简化暂时返回字符串
	return fmt.Sprintf("%d", timestamp)
}

// ExtractVideoID 提取视频ID
func ExtractVideoID(input string) (string, error) {
	// 直接匹配常见格式
	if strings.HasPrefix(input, "BV") && len(input) == 12 { // 修复：现代BV号总长度为12个字符
		return input, nil
	}
	if strings.HasPrefix(input, "av") {
		return input, nil
	}
	if strings.HasPrefix(input, "ep") {
		return input, nil
	}
	if strings.HasPrefix(input, "ss") {
		return input, nil
	}

	// 从URL中提取
	idType, id, err := ExtractFromURL(input)
	if err != nil {
		return "", err
	}

	switch idType {
	case "bv":
		return "BV" + id, nil
	case "av":
		return "av" + id, nil
	case "ep":
		return "ep" + id, nil
	case "ss":
		return "ss" + id, nil
	case "cheese_ep":
		return "cheese:" + id, nil
	case "cheese_ss":
		return "cheese:" + id, nil
	case "season":
		return "season:" + id, nil
	case "medialist":
		return "medialist:" + id, nil
	default:
		return id, nil
	}
}

// NormalizeID 标准化ID格式
func NormalizeID(id string) string {
	if strings.HasPrefix(id, "BV") {
		return id
	}
	if strings.HasPrefix(id, "av") {
		converter := NewBVConverter()
		av := strings.TrimPrefix(id, "av")
		if bv, err := converter.AVToBV(av); err == nil {
			return bv
		}
	}
	if strings.HasPrefix(id, "ep") || strings.HasPrefix(id, "ss") {
		return id
	}
	if strings.HasPrefix(id, "cheese_ep") {
		return "cheese:" + strings.TrimPrefix(id, "cheese_ep")
	}
	if strings.HasPrefix(id, "cheese_ss") {
		return "cheese:" + strings.TrimPrefix(id, "cheese_ss")
	}
	if strings.HasPrefix(id, "collection") {
		return "collection:" + strings.TrimPrefix(id, "collection")
	}
	if strings.HasPrefix(id, "series") {
		return "series:" + strings.TrimPrefix(id, "series")
	}
	return id
}
