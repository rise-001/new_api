package controller

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fullNotifySettingRequest 填满所有通知方式的字段，用来验证只有当前选中的方式会被写入设置。
func fullNotifySettingRequest(notifyType string) UpdateUserSettingRequest {
	return UpdateUserSettingRequest{
		QuotaWarningType:      notifyType,
		QuotaWarningThreshold: 500000,
		WebhookUrl:            "https://example.com/hook",
		WebhookSecret:         "webhook-secret",
		NotificationEmail:     "notify@example.com",
		BarkUrl:               "https://api.day.app/barkkey",
		GotifyUrl:             "https://gotify.example.com",
		GotifyToken:           "gotify-token",
		GotifyPriority:        7,
		WeComWebhookUrl:       "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=wecomkey",
	}
}

func TestApplyNotifySettingsOnlyWritesSelectedChannel(t *testing.T) {
	tests := []struct {
		name     string
		expected dto.UserSetting
	}{
		{
			name: dto.NotifyTypeEmail,
			expected: dto.UserSetting{
				NotifyType:        dto.NotifyTypeEmail,
				NotificationEmail: "notify@example.com",
			},
		},
		{
			name: dto.NotifyTypeWebhook,
			expected: dto.UserSetting{
				NotifyType:    dto.NotifyTypeWebhook,
				WebhookUrl:    "https://example.com/hook",
				WebhookSecret: "webhook-secret",
			},
		},
		{
			name: dto.NotifyTypeBark,
			expected: dto.UserSetting{
				NotifyType: dto.NotifyTypeBark,
				BarkUrl:    "https://api.day.app/barkkey",
			},
		},
		{
			name: dto.NotifyTypeGotify,
			expected: dto.UserSetting{
				NotifyType:     dto.NotifyTypeGotify,
				GotifyUrl:      "https://gotify.example.com",
				GotifyToken:    "gotify-token",
				GotifyPriority: 7,
			},
		},
		{
			name: dto.NotifyTypeWeCom,
			expected: dto.UserSetting{
				NotifyType:      dto.NotifyTypeWeCom,
				WeComWebhookUrl: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=wecomkey",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := fullNotifySettingRequest(tt.name)
			settings := dto.UserSetting{NotifyType: tt.name}
			applyNotifySettings(&settings, &req)
			assert.Equal(t, tt.expected, settings)
		})
	}
}

func TestApplyNotifySettingsClampsGotifyPriority(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		expected int
	}{
		{name: "below range", priority: -1, expected: 5},
		{name: "above range", priority: 11, expected: 5},
		{name: "lowest valid", priority: 0, expected: 0},
		{name: "highest valid", priority: 10, expected: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := fullNotifySettingRequest(dto.NotifyTypeGotify)
			req.GotifyPriority = tt.priority
			settings := dto.UserSetting{NotifyType: dto.NotifyTypeGotify}
			applyNotifySettings(&settings, &req)
			assert.Equal(t, tt.expected, settings.GotifyPriority)
		})
	}
}

func TestValidateUserSettingRequestWeComWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 让错误响应里出现稳定的消息 key，不依赖 i18n 是否已经初始化。
	originalTranslate := common.TranslateMessage
	common.TranslateMessage = func(c *gin.Context, key string, args ...map[string]any) string {
		return key
	}
	t.Cleanup(func() { common.TranslateMessage = originalTranslate })

	tests := []struct {
		name    string
		url     string
		wantOk  bool
		wantMsg string
	}{
		{
			name:   "valid https url",
			url:    "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=wecomkey",
			wantOk: true,
		},
		{
			name:    "empty url",
			url:     "",
			wantMsg: "setting.wecom_url_empty",
		},
		{
			name:    "not a url",
			url:     "wecomkey",
			wantMsg: "setting.wecom_url_invalid",
		},
		{
			name:    "non http scheme",
			url:     "ftp://qyapi.weixin.qq.com/send",
			wantMsg: "setting.url_must_http",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)

			req := fullNotifySettingRequest(dto.NotifyTypeWeCom)
			req.WeComWebhookUrl = tt.url

			ok := validateUserSettingRequest(c, &req)
			require.Equal(t, tt.wantOk, ok)
			if tt.wantOk {
				return
			}
			assert.True(t, strings.Contains(recorder.Body.String(), tt.wantMsg), "body: %s", recorder.Body.String())
		})
	}
}
