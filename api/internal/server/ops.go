package server

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/samuelfabel/megumi-kura/api/internal/auth"
	"github.com/samuelfabel/megumi-kura/api/internal/captcha"
	"github.com/samuelfabel/megumi-kura/api/internal/kit"
	"github.com/samuelfabel/megumi-kura/api/internal/mail"
	"github.com/samuelfabel/megumi-kura/api/internal/media"
	"github.com/samuelfabel/megumi-kura/api/internal/promise"
	"github.com/samuelfabel/megumi-kura/api/internal/quantity"
	"github.com/samuelfabel/megumi-kura/api/internal/social"
	"github.com/samuelfabel/megumi-kura/api/internal/user"
)

func (a *API) requirePerm(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		su, ok := auth.CurrentUser(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		u, err := a.Users.FindByID(c.Request.Context(), su.ID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		ok, err = a.Users.HasPermission(c.Request.Context(), u, perm)
		if err != nil || !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Set("full_user", u)
		c.Next()
	}
}

func (a *API) getPublicMedia(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ct, data, err := a.Media.GetData(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.Data(http.StatusOK, ct, data)
}

func (a *API) postAdminMedia(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}
	if file.Size > media.MaxBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file exceeds 1MB limit"})
		return
	}
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read file"})
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, media.MaxBytes+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read file"})
		return
	}
	kind := c.DefaultPostForm("kind", "other")
	ct := file.Header.Get("Content-Type")
	if ct == "" {
		ct = http.DetectContentType(data)
	}
	asset, err := a.Media.Create(c.Request.Context(), kind, file.Filename, ct, data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, asset)
}

func (a *API) listPublicKits(c *gin.Context) {
	items, err := a.Kits.List(c.Request.Context(), true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list kits"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"kits": items})
}

func (a *API) listAdminKits(c *gin.Context) {
	items, err := a.Kits.List(c.Request.Context(), false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list kits"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"kits": items})
}

func (a *API) createKit(c *gin.Context) {
	var in kit.CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	k, err := a.Kits.Create(c.Request.Context(), in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, k)
}

func (a *API) updateKit(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var body struct {
		kit.CreateInput
		Active *bool `json:"active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	k, err := a.Kits.Update(c.Request.Context(), id, body.CreateInput, body.Active)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, k)
}

func (a *API) putKitItems(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var body struct {
		Items []kit.PutItem `json:"items"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	k, err := a.Kits.ReplaceItems(c.Request.Context(), id, body.Items)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, k)
}

func (a *API) createPublicPromiseBatch(c *gin.Context) {
	var body struct {
		promise.BatchInput
		CaptchaToken  string `json:"captcha_token"`
		CaptchaAnswer string `json:"captcha_answer"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if a.Captcha == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "captcha unavailable"})
		return
	}
	if err := a.Captcha.Verify(c.Request.Context(), body.CaptchaToken, body.CaptchaAnswer, c.ClientIP()); err != nil {
		switch {
		case errors.Is(err, captcha.ErrExpired):
			c.JSON(http.StatusBadRequest, gin.H{"error": "captcha expired"})
		case errors.Is(err, captcha.ErrUsed):
			c.JSON(http.StatusBadRequest, gin.H{"error": "captcha already used"})
		case errors.Is(err, captcha.ErrMisconfigured):
			c.JSON(http.StatusInternalServerError, gin.H{"error": "captcha unavailable"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid captcha"})
		}
		return
	}
	in := body.BatchInput
	if in.KitID != nil && *in.KitID > 0 {
		expanded, err := a.Kits.ExpandItems(c.Request.Context(), *in.KitID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		for _, it := range expanded {
			in.Items = append(in.Items, promise.BatchItemInput{FoodID: it.FoodID, Quantity: it.Quantity})
		}
	}
	res, err := a.Promises.CreateBatch(c.Request.Context(), in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (a *API) getProjectionSeries(c *gin.Context) {
	foodID, _ := strconv.ParseInt(c.Query("food_id"), 10, 64)
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if foodID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "food_id required"})
		return
	}
	series, err := a.Projections.TimeSeries(c.Request.Context(), foodID, days)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, series)
}

func (a *API) getShortageRisks(c *gin.Context) {
	risks, err := a.Projections.ShortageRisks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load risks"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"risks": risks})
}

func (a *API) subscribeMail(c *gin.Context) {
	var body struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	res, err := a.Mail.Subscribe(c.Request.Context(), body.Email)
	if err != nil {
		// Still avoid revealing too much on invalid; return generic bad request
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (a *API) unsubscribeMail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		var body struct {
			Token string `json:"token"`
		}
		_ = c.ShouldBindJSON(&body)
		token = body.Token
	}
	if err := a.Mail.Unsubscribe(c.Request.Context(), token); err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": true}) // soft success
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *API) listMailTemplates(c *gin.Context) {
	items, err := a.Mail.ListTemplates(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"templates": items})
}

func (a *API) updateMailTemplate(c *gin.Context) {
	key := c.Param("key")
	var body struct {
		Subject  string `json:"subject"`
		BodyHTML string `json:"body_html"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	t, err := a.Mail.UpdateTemplate(c.Request.Context(), key, body.Subject, body.BodyHTML)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, t)
}

func (a *API) getMailSettings(c *gin.Context) {
	cfg, _ := a.Mail.LoadSMTP(c.Request.Context())
	cfg.Password = "" // never echo password
	alerts := a.Mail.LoadAlertSettings(c.Request.Context())
	count, _ := a.Mail.ActiveSubscriberCount(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{
		"smtp":             cfg,
		"alerts":           alerts,
		"subscriber_count": count,
		// emails themselves are NEVER listed
	})
}

func (a *API) putMailSettings(c *gin.Context) {
	var body struct {
		SMTP   mail.SMTPConfig    `json:"smtp"`
		Alerts mail.AlertSettings `json:"alerts"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	// Keep existing password if blank
	existing, _ := a.Mail.LoadSMTP(c.Request.Context())
	if body.SMTP.Password == "" {
		body.SMTP.Password = existing.Password
	}
	if err := a.Mail.SaveSMTP(c.Request.Context(), body.SMTP); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := a.Mail.SaveAlertSettings(c.Request.Context(), body.Alerts); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	a.getMailSettings(c)
}

func (a *API) sendShortageMail(c *gin.Context) {
	alerts := a.Mail.LoadAlertSettings(c.Request.Context())
	shortages, err := a.Mail.FindShortages(c.Request.Context(), alerts.GlobalThreshold)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
		return
	}
	site, _ := a.Settings.Public(c.Request.Context())
	base := c.Request.Header.Get("Origin")
	if base == "" {
		base = "http://" + c.Request.Host
	}
	res, err := a.Mail.SendShortageAlert(c.Request.Context(), site.Name, base, "manual", shortages)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (a *API) putFoodThreshold(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var body struct {
		Threshold quantity.Q `json:"threshold"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := a.Mail.SetFoodThreshold(c.Request.Context(), id, body.Threshold); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *API) listSocialPublic(c *gin.Context) {
	posts, err := a.Social.ListPosts(c.Request.Context(), true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"posts": posts})
}

func (a *API) getSocialPublic(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	p, err := a.Social.GetPost(c.Request.Context(), id, true)
	if err != nil || !p.Published {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, p)
}

func (a *API) commentSocialPublic(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var in social.CommentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	cm, err := a.Social.AddComment(c.Request.Context(), id, in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": cm.ID, "status": cm.Status})
}

func (a *API) listSocialAdmin(c *gin.Context) {
	posts, err := a.Social.ListPosts(c.Request.Context(), false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
		return
	}
	pending, _ := a.Social.ListPendingComments(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"posts": posts, "pending_comments": pending})
}

func (a *API) createSocialAdmin(c *gin.Context) {
	var in social.PostInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	su, _ := auth.CurrentUser(c)
	uid := su.ID
	p, err := a.Social.CreatePost(c.Request.Context(), in, &uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (a *API) updateSocialAdmin(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var in social.PostInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	p, err := a.Social.UpdatePost(c.Request.Context(), id, in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}

func (a *API) moderateComment(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var body struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	su, _ := auth.CurrentUser(c)
	uid := su.ID
	cm, err := a.Social.ModerateComment(c.Request.Context(), id, body.Status, &uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cm)
}

func (a *API) listUsers(c *gin.Context) {
	items, err := a.Users.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": items, "all_permissions": user.AllPermissions})
}

func (a *API) createUser(c *gin.Context) {
	var in user.CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	u, err := a.Users.Create(c.Request.Context(), in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, u)
}

func (a *API) putUserPermissions(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var body struct {
		Permissions []string `json:"permissions"`
		Active      *bool    `json:"active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := a.Users.ReplacePermissions(c.Request.Context(), id, body.Permissions); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Active != nil {
		_ = a.Users.SetActive(c.Request.Context(), id, *body.Active)
	}
	u, err := a.Users.FindByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, u)
}

func (a *API) requestPasswordReset(c *gin.Context) {
	var body struct {
		Email string `json:"email"`
	}
	_ = c.ShouldBindJSON(&body)
	// Always ok — no enumeration
	u, err := a.Users.FindByEmail(c.Request.Context(), body.Email)
	if err == nil && !u.IsBootstrapAdmin && u.Email != nil {
		raw, err := a.Users.CreateResetToken(c.Request.Context(), u.ID, 2*time.Hour)
		if err == nil {
			tpl, terr := a.Mail.GetTemplate(c.Request.Context(), "password_reset")
			cfg, _ := a.Mail.LoadSMTP(c.Request.Context())
			if terr == nil && cfg.Host != "" {
				site, _ := a.Settings.Public(c.Request.Context())
				base := c.Request.Header.Get("Origin")
				if base == "" {
					base = "http://" + c.Request.Host
				}
				vars := map[string]string{
					"site_name":    site.Name,
					"display_name": u.DisplayName,
					"reset_url":    strings.TrimRight(base, "/") + "/#/reset?token=" + raw,
				}
				_ = sendPasswordReset(cfg, *u.Email, mail.Render(tpl.Subject, vars), mail.Render(tpl.BodyHTML, vars))
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func sendPasswordReset(cfg mail.SMTPConfig, to, subject, body string) error {
	// reuse package-level via a minimal wrapper — call FindShortages path is heavy;
	// for simplicity duplicate is avoided: use mail package SendShortage only.
	// Password reset uses internal unexported send — expose via exported helper.
	return mail.SendHTML(cfg, to, subject, body)
}

func (a *API) resetPassword(c *gin.Context) {
	var body struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Token == "" || body.NewPassword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := a.Users.ConsumeResetToken(c.Request.Context(), body.Token, body.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *API) createDeliveryWithKit(c *gin.Context) {
	a.createDelivery(c)
}

func (a *API) setFoodIcon(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var body struct {
		IconMediaID *int64 `json:"icon_media_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := a.Foods.SetIcon(c.Request.Context(), id, body.IconMediaID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
