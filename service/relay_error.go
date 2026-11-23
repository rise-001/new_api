package service

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
)

// ClientErrorMessage formats the message sent to an API client. A match always
// carries the gateway request ID, including on protocols that normally omit it.
func ClientErrorMessage(c *gin.Context, message string, withRequestID bool) (string, bool) {
	matched := operation_setting.MatchesClientError(message)
	if matched {
		lang := i18n.StatedLang(c)
		if lang == "" && common.DefaultLanguage == "" {
			lang = i18n.LangZhCN
		}
		message = i18n.Translate(lang, i18n.MsgRelayMaskedClientError)
	}
	if matched || withRequestID {
		message = common.MessageWithRequestId(message, c.GetString(common.RequestIdKey))
	}
	return message, matched
}

// ClientRelayError keeps the upstream error available to logs and report hooks.
// The protocol converters can read either Err or the upstream protocol's DTO.
func ClientRelayError(c *gin.Context, err *types.NewAPIError, withRequestID bool) *types.NewAPIError {
	if err == nil {
		return nil
	}
	message, masked := ClientErrorMessage(c, err.Error(), withRequestID)
	if !masked && !withRequestID {
		return err
	}
	client := *err
	client.SetMessage(message)
	if masked {
		client.Metadata = nil
		switch payload := client.RelayError.(type) {
		case types.OpenAIError:
			payload.Message, payload.Metadata, payload.Param = message, nil, ""
			client.RelayError = payload
		case types.ClaudeError:
			payload.Message = message
			client.RelayError = payload
		}
	}
	return &client
}

// DecideRelayRetry is the single retry decision for relay attempts. The reason
// is recorded in the request policy decision events of the log details.
func DecideRelayRetry(c *gin.Context, err *types.NewAPIError, retryTimes int) PolicyDecision {
	if err == nil {
		return PolicyDecision{Action: "stop", Reason: "request_completed", Source: "system"}
	}
	if ShouldSkipRetryAfterChannelAffinityFailure(c) {
		source := RequestPolicy(c).SessionModeSource
		if source == "" {
			source = "session_rule"
		}
		return PolicyDecision{Action: "stop", Reason: "strict_session", Source: source}
	}
	if GetChannelConstraints(c).SuppressesRetry() {
		return PolicyDecision{Action: "stop", Reason: "pinned_channel", Source: "channel_constraint"}
	}
	if types.IsChannelError(err) {
		return PolicyDecision{Action: "retry", Reason: "channel_error", Source: "system"}
	}
	if types.IsSkipRetryError(err) {
		return PolicyDecision{Action: "stop", Reason: "non_retryable_error", Source: "system"}
	}
	if retryTimes <= 0 {
		return PolicyDecision{Action: "stop", Reason: "attempt_budget_exhausted", Source: "global"}
	}
	code := err.StatusCode
	if code >= 200 && code < 300 {
		return PolicyDecision{Action: "stop", Reason: "system_retry_exclusion", Source: "system"}
	}
	if code < 100 || code > 599 {
		return PolicyDecision{Action: "retry", Reason: "unrecognized_status", Source: "system"}
	}
	if operation_setting.IsAlwaysSkipRetryCode(err.GetErrorCode()) || operation_setting.IsAlwaysSkipRetryStatusCode(code) {
		return PolicyDecision{Action: "stop", Reason: "system_retry_exclusion", Source: "system"}
	}
	if operation_setting.ShouldRetryByStatusCode(code) {
		return PolicyDecision{Action: "retry", Reason: "retry_status_matched", Source: "global"}
	}
	return PolicyDecision{Action: "stop", Reason: "status_not_retryable", Source: "global"}
}

func ShouldRetryRelayError(c *gin.Context, openaiErr *types.NewAPIError, retryTimes int) bool {
	return DecideRelayRetry(c, openaiErr, retryTimes).Action == "retry"
}

func ProcessChannelError(c *gin.Context, channelError types.ChannelError, err *types.NewAPIError, relayInfo *relaycommon.RelayInfo) {
	if err == nil {
		return
	}
	upstreamURLs := common.UpstreamAddresses(c)
	logger.LogError(c, common.LogText("channel error (channel #%d, status code: %d): %s", channelError.ChannelId, err.StatusCode, common.LocalLogPreview(err.MaskSensitiveErrorWithStatusCode(upstreamURLs...))))
	if ShouldDisableChannel(err) && channelError.AutoBan {
		reason := err.MaskSensitiveErrorWithStatusCode(upstreamURLs...)
		gopool.Go(func() {
			DisableChannel(channelError, reason)
		})
	}

	RecordRelayErrorLog(c, channelError.ChannelId, err, relayInfo)
}

// RecordRelayErrorLog writes the error log of a failed relay request when
// error logs are on. channelId is 0 for a request that reached no channel.
func RecordRelayErrorLog(c *gin.Context, channelId int, err *types.NewAPIError, relayInfo *relaycommon.RelayInfo) {
	if !constant.ErrorLogEnabled || !types.IsRecordErrorLog(err) {
		return
	}
	userId := c.GetInt("id")
	tokenName := c.GetString("token_name")
	modelName := c.GetString("original_model")
	tokenId := c.GetInt("token_id")
	userGroup := common.GetContextKeyString(c, constant.ContextKeyUsingGroup)
	if userGroup == "auto" {
		if selected := common.GetContextKeyString(c, constant.ContextKeyAutoGroup); selected != "" {
			userGroup = selected
		}
	}
	other := model.NewLogOther()
	if c.Request != nil && c.Request.URL != nil {
		other.SetPublic("request_path", c.Request.URL.Path)
	}
	other.SetPublic("error_type", err.GetErrorType())
	other.SetPublic("error_code", err.GetErrorCode())
	other.SetPublic("status_code", err.StatusCode)
	AppendRelayLogAdminInfo(c, relayInfo, other)
	appendRequestPluginRecord(relayInfo, other)
	AppendResponseModelLogInfo(relayInfo, other)
	AppendTaskPluginContextAuditInfo(c, other)
	startTime := common.GetContextKeyTime(c, constant.ContextKeyRequestStartTime)
	if startTime.IsZero() {
		startTime = time.Now()
	}
	useTimeSeconds := int(time.Since(startTime).Seconds())
	model.RecordErrorLog(c, userId, channelId, modelName, tokenName, err.MaskSensitiveErrorWithStatusCode(common.UpstreamAddresses(c)...), tokenId, useTimeSeconds, common.GetContextKeyBool(c, constant.ContextKeyIsStream), userGroup, other)
}
