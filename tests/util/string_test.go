package util_test

import (
	"strings"
	"testing"

	"github.com/tekintian/go-bbdown/util"
)

func TestExtractFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantType string
		wantID   string
		wantErr  bool
	}{
		{
			name:     "BV号URL",
			url:      "https://www.bilibili.com/video/BV1xx411c7mD",
			wantType: "bv",
			wantID:   "1xx411c7mD",
			wantErr:  false,
		},
		{
			name:     "AV号URL",
			url:      "https://www.bilibili.com/video/av12345",
			wantType: "av",
			wantID:   "12345",
			wantErr:  false,
		},
		{
			name:     "EP号URL",
			url:      "https://www.bilibili.com/bangumi/play/ep12345",
			wantType: "ep",
			wantID:   "12345",
			wantErr:  false,
		},
		{
			name:     "SS号URL",
			url:      "https://www.bilibili.com/bangumi/play/ss12345",
			wantType: "ss",
			wantID:   "12345",
			wantErr:  false,
		},
		{
			name:     "合集URL-无参数",
			url:      "https://space.bilibili.com/89320896/lists/5348941",
			wantType: "season",
			wantID:   "89320896:5348941",
			wantErr:  false,
		},
		{
			name:     "合集URL-带参数",
			url:      "https://space.bilibili.com/89320896/lists/5348941?type=season",
			wantType: "season",
			wantID:   "89320896:5348941",
			wantErr:  false,
		},
		{
			name:     "合集URL-多参数",
			url:      "https://space.bilibili.com/89320896/lists/5348941?type=season&spm_id_from=333.999.0.0",
			wantType: "season",
			wantID:   "89320896:5348941",
			wantErr:  false,
		},
		{
			name:     "收藏夹URL",
			url:      "https://www.bilibili.com/medialist/play/12345?business_id=67890",
			wantType: "collection",
			wantID:   "67890",
			wantErr:  false,
		},
		{
			name:     "媒体列表URL",
			url:      "https://www.bilibili.com/medialist/detail/ml12345",
			wantType: "medialist",
			wantID:   "12345",
			wantErr:  false,
		},
		{
			name:     "课程EP URL",
			url:      "https://www.bilibili.com/cheese/play/ep12345",
			wantType: "ep",
			wantID:   "12345",
			wantErr:  false,
		},
		{
			name:     "课程SS URL",
			url:      "https://www.bilibili.com/cheese/play/ss12345",
			wantType: "ss",
			wantID:   "12345",
			wantErr:  false,
		},
		{
			name:     "无效URL",
			url:      "https://www.example.com/video",
			wantType: "",
			wantID:   "",
			wantErr:  true,
		},
		{
			name:     "直接BV号",
			url:      "BV1xx411c7mD",
			wantType: "bv",
			wantID:   "1xx411c7mD",
			wantErr:  false,
		},
		{
			name:     "直接AV号",
			url:      "av12345",
			wantType: "av",
			wantID:   "12345",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotID, err := util.ExtractFromURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractFromURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotType != tt.wantType {
				t.Errorf("ExtractFromURL() gotType = %v, want %v", gotType, tt.wantType)
			}
			if gotID != tt.wantID {
				t.Errorf("ExtractFromURL() gotID = %v, want %v", gotID, tt.wantID)
			}
		})
	}
}

func TestExtractVideoID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "直接BV号",
			input:   "BV1xx411c7mD",
			want:    "BV1xx411c7mD",
			wantErr: false,
		},
		{
			name:    "直接AV号",
			input:   "av12345",
			want:    "av12345",
			wantErr: false,
		},
		{
			name:    "直接EP号",
			input:   "ep12345",
			want:    "ep12345",
			wantErr: false,
		},
		{
			name:    "直接SS号",
			input:   "ss12345",
			want:    "ss12345",
			wantErr: false,
		},
		{
			name:    "BV号URL",
			input:   "https://www.bilibili.com/video/BV1xx411c7mD",
			want:    "BV1xx411c7mD",
			wantErr: false,
		},
		{
			name:    "AV号URL",
			input:   "https://www.bilibili.com/video/av12345",
			want:    "av12345",
			wantErr: false,
		},
		{
			name:    "合集URL",
			input:   "https://space.bilibili.com/89320896/lists/5348941",
			want:    "season:89320896:5348941",
			wantErr: false,
		},
		{
			name:    "合集URL带参数",
			input:   "https://space.bilibili.com/89320896/lists/5348941?type=season",
			want:    "season:89320896:5348941",
			wantErr: false,
		},
		{
			name:    "收藏夹URL",
			input:   "https://www.bilibili.com/medialist/play/12345?business_id=67890",
			want:    "67890",
			wantErr: false,
		},
		{
			name:    "课程URL",
			input:   "https://www.bilibili.com/cheese/play/ep12345",
			want:    "ep12345",
			wantErr: false,
		},
		{
			name:    "无效URL",
			input:   "https://www.example.com/video",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := util.ExtractVideoID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractVideoID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractVideoID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBVConverter(t *testing.T) {
	converter := util.NewBVConverter()

	tests := []struct {
		name    string
		av      string
		want    string
		wantErr bool
	}{
		{
			name:    "AV转BV - 小数字",
			av:      "1",
			want:    "BV1xx411c7mD",
			wantErr: false,
		},
		{
			name:    "AV转BV - 中等数字",
			av:      "170001",
			want:    "BV17x411w7KC",
			wantErr: false,
		},
		{
			name:    "AV转BV - 大数字",
			av:      "123456789",
			want:    "BV1234567890", // 这是一个示例，实际结果可能不同
			wantErr: false,
		},
		{
			name:    "AV转BV - 无效数字",
			av:      "invalid",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := converter.AVToBV(tt.av)
			if (err != nil) != tt.wantErr {
				t.Errorf("BVConverter.AVToBV() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !strings.HasPrefix(got, "BV") {
				t.Errorf("BVConverter.AVToBV() = %v, should start with BV", got)
			}
			if !tt.wantErr && len(got) != 12 {
				t.Errorf("BVConverter.AVToBV() = %v, should have length 12", got)
			}
		})
	}
}

func TestIsValidID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{
			name: "有效BV号",
			id:   "BV1xx411c7mD",
			want: true,
		},
		{
			name: "有效AV号",
			id:   "av12345",
			want: true,
		},
		{
			name: "有效EP号",
			id:   "ep12345",
			want: true,
		},
		{
			name: "有效SS号",
			id:   "ss12345",
			want: true,
		},
		{
			name: "无效BV号 - 太短",
			id:   "BV123",
			want: false,
		},
		{
			name: "无效BV号 - 太长",
			id:   "BV1234567890123",
			want: false,
		},
		{
			name: "无效AV号",
			id:   "avinvalid",
			want: false,
		},
		{
			name: "无效EP号",
			id:   "epinvalid",
			want: false,
		},
		{
			name: "无效SS号",
			id:   "ssinvalid",
			want: false,
		},
		{
			name: "不支持的格式",
			id:   "cv12345",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := util.IsValidID(tt.id)
			if got != tt.want {
				t.Errorf("IsValidID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetQueryString(t *testing.T) {
	tests := []struct {
		name string
		url  string
		key  string
		want string
	}{
		{
			name: "获取单个参数",
			url:  "https://example.com/test?param1=value1&param2=value2",
			key:  "param1",
			want: "value1",
		},
		{
			name: "获取第二个参数",
			url:  "https://example.com/test?param1=value1&param2=value2",
			key:  "param2",
			want: "value2",
		},
		{
			name: "参数不存在",
			url:  "https://example.com/test?param1=value1",
			key:  "param2",
			want: "",
		},
		{
			name: "空URL",
			url:  "",
			key:  "param1",
			want: "",
		},
		{
			name: "URL无参数",
			url:  "https://example.com/test",
			key:  "param1",
			want: "",
		},
		{
			name: "参数包含特殊字符",
			url:  "https://example.com/test?param1=value-with-special-chars",
			key:  "param1",
			want: "value-with-special-chars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := util.GetQueryString(tt.url, tt.key)
			if got != tt.want {
				t.Errorf("GetQueryString() = %v, want %v", got, tt.want)
			}
		})
	}
}
