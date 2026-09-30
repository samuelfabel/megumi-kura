package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/samuelfabel/megumi-kura/api/internal/auth"
	"github.com/samuelfabel/megumi-kura/api/internal/category"
	"github.com/samuelfabel/megumi-kura/api/internal/delivery"
	"github.com/samuelfabel/megumi-kura/api/internal/need"
	"github.com/samuelfabel/megumi-kura/api/internal/person"
	"github.com/samuelfabel/megumi-kura/api/internal/stock"
)

func (a *API) listCategories(c *gin.Context) {
	items, err := a.Categories.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list categories"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"categories": items})
}

func (a *API) createCategory(c *gin.Context) {
	var in category.CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	item, err := a.Categories.Create(c.Request.Context(), in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (a *API) updateCategory(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var in category.CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	item, err := a.Categories.Update(c.Request.Context(), id, in)
	if err != nil {
		if errors.Is(err, category.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (a *API) listCommunityNeeds(c *gin.Context) {
	items, err := a.Needs.List(c.Request.Context(), false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list needs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"needs": items})
}

func (a *API) createCommunityNeed(c *gin.Context) {
	var in need.CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	item, err := a.Needs.Create(c.Request.Context(), in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (a *API) deleteCommunityNeed(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := a.Needs.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, need.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *API) listPeople(c *gin.Context) {
	items, err := a.People.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list people"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"people": items})
}

func (a *API) createPerson(c *gin.Context) {
	var in person.CreatePersonInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	item, err := a.People.Create(c.Request.Context(), in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (a *API) updatePerson(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body struct {
		person.CreatePersonInput
		Active *bool `json:"active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	active := true
	if body.Active != nil {
		active = *body.Active
	}
	item, err := a.People.Update(c.Request.Context(), id, body.CreatePersonInput, active)
	if err != nil {
		if errors.Is(err, person.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (a *API) listPersonNeeds(c *gin.Context) {
	var personID *int64
	if v := c.Query("person_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid person_id"})
			return
		}
		personID = &id
	}
	items, err := a.People.ListNeeds(c.Request.Context(), personID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list person needs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"needs": items})
}

func (a *API) createPersonNeed(c *gin.Context) {
	var in person.CreateNeedInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	item, err := a.People.CreateNeed(c.Request.Context(), in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (a *API) deletePersonNeed(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := a.People.DeleteNeed(c.Request.Context(), id); err != nil {
		if errors.Is(err, person.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *API) listDeliveries(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := a.Deliveries.List(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list deliveries"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deliveries": items})
}

func (a *API) createDelivery(c *gin.Context) {
	var in delivery.CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	u, _ := auth.CurrentUser(c)
	uid := u.ID
	item, err := a.Deliveries.Create(c.Request.Context(), in, &uid)
	if err != nil {
		if errors.Is(err, stock.ErrInsufficientStock) {
			c.JSON(http.StatusConflict, gin.H{"error": "insufficient stock"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}
