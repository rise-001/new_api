package controller

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/oauth"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupUserRegistrationIPTest(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupManageUserTestDB(t)
	previousRegister, previousPasswordRegister := common.RegisterEnabled, common.PasswordRegisterEnabled
	previousEmailVerification := common.EmailVerificationEnabled
	previousGenerateToken := constant.GenerateDefaultToken
	previousQuota := common.QuotaForNewUser
	common.RegisterEnabled, common.PasswordRegisterEnabled = true, true
	common.EmailVerificationEnabled = false
	constant.GenerateDefaultToken = false
	common.QuotaForNewUser = 0
	t.Cleanup(func() {
		common.RegisterEnabled, common.PasswordRegisterEnabled = previousRegister, previousPasswordRegister
		common.EmailVerificationEnabled = previousEmailVerification
		constant.GenerateDefaultToken = previousGenerateToken
		common.QuotaForNewUser = previousQuota
	})
	return db
}

func TestRegisterRecordsClientIPInsteadOfSubmittedIP(t *testing.T) {
	for _, ip := range []string{"203.0.113.8", "2001:db8:abcd:1234:5678:9abc:def0:1234"} {
		t.Run(ip, func(t *testing.T) {
			db := setupUserRegistrationIPTest(t)
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader(
				`{"username":"registered-user","password":"password123","register_ip":"198.51.100.99"}`,
			))
			c.Request.RemoteAddr = net.JoinHostPort(ip, "12345")

			Register(c)

			require.Equal(t, http.StatusOK, recorder.Code)
			var response struct {
				Success bool `json:"success"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			require.True(t, response.Success, recorder.Body.String())
			var user model.User
			require.NoError(t, db.First(&user, "username = ?", "registered-user").Error)
			assert.Equal(t, ip, user.RegisterIP)

			user.RegisterIP = "198.51.100.99"
			user.DisplayName = "Updated name"
			require.NoError(t, user.Update(false))
			require.NoError(t, db.First(&user, user.Id).Error)
			assert.Equal(t, "Updated name", user.DisplayName)
			assert.Equal(t, ip, user.RegisterIP, "profile updates must preserve the registration IP")
		})
	}
}

func TestOAuthRegistrationIPSurvivesLaterLogin(t *testing.T) {
	for _, tt := range []struct {
		name     string
		provider oauth.Provider
	}{
		{name: "built-in", provider: &oauth.GitHubProvider{}},
		{name: "custom", provider: oauth.NewGenericOAuthProvider(&model.CustomOAuthProvider{
			Id: 1, Name: "Custom provider", Slug: "custom",
		})},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db := setupUserRegistrationIPTest(t)
			require.NoError(t, db.AutoMigrate(&model.UserOAuthBinding{}))
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodGet, "/api/oauth/provider", nil)
			c.Request.RemoteAddr = "[2001:db8::42]:12345"
			oauthUser := &oauth.OAuthUser{ProviderUserID: "external-user", Username: "oauth-user"}

			user, err := findOrCreateOAuthUser(c, tt.provider, oauthUser, "")
			require.NoError(t, err)
			var stored model.User
			require.NoError(t, db.First(&stored, user.Id).Error)
			assert.Equal(t, "2001:db8::42", stored.RegisterIP)

			c.Request.RemoteAddr = "198.51.100.10:12345"
			returningUser, err := findOrCreateOAuthUser(c, tt.provider, oauthUser, "")
			require.NoError(t, err)
			assert.Equal(t, user.Id, returningUser.Id)
			assert.Equal(t, "2001:db8::42", returningUser.RegisterIP)
			require.NoError(t, db.First(&stored, user.Id).Error)
			assert.Equal(t, "2001:db8::42", stored.RegisterIP)
		})
	}
}

func TestWeChatRegistrationRecordsClientIP(t *testing.T) {
	db := setupUserRegistrationIPTest(t)
	previousEnabled, previousAddress := common.WeChatAuthEnabled, common.WeChatServerAddress
	common.WeChatAuthEnabled = true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":"wechat-user-id"}`))
	}))
	defer server.Close()
	common.WeChatServerAddress = server.URL
	t.Cleanup(func() {
		common.WeChatAuthEnabled, common.WeChatServerAddress = previousEnabled, previousAddress
	})
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/oauth/wechat?code=test-code", nil)
	c.Request.RemoteAddr = "203.0.113.42:12345"

	WeChatAuth(c)

	var user model.User
	require.NoError(t, db.First(&user, "wechat_id = ?", "wechat-user-id").Error)
	assert.Equal(t, "203.0.113.42", user.RegisterIP)
	assert.Contains(t, recorder.Body.String(), `"success":true`)
}

func TestUserListResponsesIncludeRegistrationIPAndLegacyEmptyValue(t *testing.T) {
	db := setupUserRegistrationIPTest(t)
	require.NoError(t, db.Create(&model.User{
		Username: "registered-user", Password: "unused", AffCode: "new", RegisterIP: "203.0.113.8",
	}).Error)
	require.NoError(t, db.Create(&model.User{
		Username: "legacy-user", Password: "unused", AffCode: "old",
	}).Error)
	// Existing rows have NULL after the registration IP column is added.
	require.NoError(t, db.Table("users").Where("username = ?", "legacy-user").Update("register_ip", nil).Error)

	for _, tt := range []struct {
		path    string
		handler gin.HandlerFunc
	}{
		{path: "/api/user/", handler: GetAllUsers},
		{path: "/api/user/search?keyword=user", handler: SearchUsers},
	} {
		t.Run(tt.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, tt.path, nil)
			tt.handler(c)
			require.Equal(t, http.StatusOK, recorder.Code)
			var response struct {
				Success bool `json:"success"`
				Data    struct {
					Items []struct {
						Username   string  `json:"username"`
						RegisterIP *string `json:"register_ip"`
					} `json:"items"`
				} `json:"data"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			require.True(t, response.Success)
			require.Len(t, response.Data.Items, 2)
			ips := make(map[string]string)
			for _, user := range response.Data.Items {
				require.NotNil(t, user.RegisterIP)
				ips[user.Username] = *user.RegisterIP
			}
			assert.Equal(t, map[string]string{"registered-user": "203.0.113.8", "legacy-user": ""}, ips)
		})
	}
}
