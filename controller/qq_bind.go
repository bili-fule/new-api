package controller

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

func GetQqBindCode(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": common.TranslateMessage(c, i18n.MsgInvalidParams),
		})
		return
	}

	astrBotURL := common.QQBotBaseURL + "/api/plug/api/v1/bind/code?user_id=" + fmt.Sprintf("newapi_%d", userId)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(astrBotURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": common.TranslateMessage(c, i18n.MsgGenerateFailed),
		})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": common.TranslateMessage(c, i18n.MsgGenerateFailed),
		})
		return
	}

	var result struct {
		Code          string `json:"code"`
		ExpireSeconds int    `json:"expire_seconds"`
	}
	if err := common.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": common.TranslateMessage(c, i18n.MsgGenerateFailed),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func ConfirmQqBind(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": common.TranslateMessage(c, i18n.MsgInvalidParams),
		})
		return
	}

	astrBotURL := common.QQBotBaseURL + "/api/plug/api/v1/bind/query?user_id=" + fmt.Sprintf("newapi_%d", userId)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(astrBotURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": common.TranslateMessage(c, i18n.MsgOperationFailed),
		})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": common.TranslateMessage(c, i18n.MsgOperationFailed),
		})
		return
	}

	var result struct {
		Bound  bool   `json:"bound"`
		QQ     string `json:"qq"`
		UserID string `json:"user_id"`
	}
	if err := common.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": common.TranslateMessage(c, i18n.MsgOperationFailed),
		})
		return
	}

	expectedUserID := fmt.Sprintf("newapi_%d", userId)
	if result.UserID != expectedUserID && result.UserID != "" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": common.TranslateMessage(c, i18n.MsgForbidden),
		})
		return
	}

	if !result.Bound {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": common.TranslateMessage(c, i18n.MsgOperationFailed),
		})
		return
	}

	user, err := model.GetUserById(userId, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": common.TranslateMessage(c, i18n.MsgDatabaseError),
		})
		return
	}

	user.QQId = result.QQ
	if err := user.Update(false); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": common.TranslateMessage(c, i18n.MsgDatabaseError),
		})
		return
	}

	model.InvalidateUserCache(userId)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": common.TranslateMessage(c, i18n.MsgOperationSuccess),
		"data": gin.H{
			"qq": result.QQ,
		},
	})
}
