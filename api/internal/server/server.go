package server

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/samuelfabel/megumi-kura/api/internal/auth"
	"github.com/samuelfabel/megumi-kura/api/internal/basket"
	"github.com/samuelfabel/megumi-kura/api/internal/category"
	"github.com/samuelfabel/megumi-kura/api/internal/config"
	"github.com/samuelfabel/megumi-kura/api/internal/delivery"
	"github.com/samuelfabel/megumi-kura/api/internal/food"
	"github.com/samuelfabel/megumi-kura/api/internal/need"
	"github.com/samuelfabel/megumi-kura/api/internal/person"
	"github.com/samuelfabel/megumi-kura/api/internal/promise"
	"github.com/samuelfabel/megumi-kura/api/internal/quantity"
	"github.com/samuelfabel/megumi-kura/api/internal/settings"
	"github.com/samuelfabel/megumi-kura/api/internal/stock"
)

type API struct {
	Cfg        config.Config
	Auth       *auth.Service
	Foods      *food.Repository
	Stock      *stock.Repository
	Promises   *promise.Repository
	Basket     *basket.Repository
	Settings   *settings.Repository
	Categories *category.Repository
	Needs      *need.Repository
	People     *person.Repository
	Deliveries *delivery.Repository
}

func (a *API) Router() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	v1 := r.Group("/api/v1")
	{
		pub := v1.Group("/public")
		{
			pub.GET("/summary", a.getPublicSummary)
			pub.GET("/stock", a.getPublicStock)
			pub.GET("/promises", a.getPublicPromises)
			pub.POST("/promises", a.createPublicPromise)
			pub.GET("/settings", a.getPublicSettings)
			pub.GET("/needs", a.getPublicNeeds)
		}

		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/login", a.postLogin)
			authGroup.POST("/logout", a.postLogout)
			authGroup.GET("/me", a.Auth.Middleware(), a.getMe)
		}

		admin := v1.Group("/admin", a.Auth.Middleware())
		{
			admin.GET("/foods", a.listFoods)
			admin.POST("/foods", a.createFood)
			admin.POST("/stock/in", a.stockIn)
			admin.POST("/stock/out", a.stockOut)
			admin.GET("/stock/movements", a.listMovements)
			admin.GET("/promises", a.listPromises)
			admin.POST("/promises", a.createPromise)
			admin.DELETE("/promises/:id", a.deletePromise)
			admin.GET("/basket", a.getBasket)
			admin.PUT("/basket", a.putBasket)

			admin.GET("/categories", a.listCategories)
			admin.POST("/categories", a.createCategory)
			admin.PUT("/categories/:id", a.updateCategory)

			admin.GET("/community-needs", a.listCommunityNeeds)
			admin.POST("/community-needs", a.createCommunityNeed)
			admin.DELETE("/community-needs/:id", a.deleteCommunityNeed)

			admin.GET("/people", a.listPeople)
			admin.POST("/people", a.createPerson)
			admin.PUT("/people/:id", a.updatePerson)
			admin.GET("/person-needs", a.listPersonNeeds)
			admin.POST("/person-needs", a.createPersonNeed)
			admin.DELETE("/person-needs/:id", a.deletePersonNeed)

			admin.GET("/deliveries", a.listDeliveries)
			admin.POST("/deliveries", a.createDelivery)
		}
	}

	a.mountStatic(r)
	return r
}

func (a *API) mountStatic(r *gin.Engine) {
	staticDir := a.Cfg.StaticDir
	if staticDir == "" {
		return
	}
	abs, err := filepath.Abs(staticDir)
	if err != nil {
		return
	}
	if st, err := os.Stat(abs); err != nil || !st.IsDir() {
		r.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"name":    "Megumi Kura",
				"message": "API is running. Build the frontend (make web-build) to serve the dashboard.",
			})
		})
		return
	}

	r.Static("/assets", filepath.Join(abs, "assets"))
	r.StaticFile("/favicon.svg", filepath.Join(abs, "favicon.svg"))
	r.NoRoute(func(c *gin.Context) {
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.File(filepath.Join(abs, "index.html"))
	})
}

type needProgress struct {
	FoodID      int64      `json:"food_id"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Unit        string     `json:"unit"`
	Needed      quantity.Q `json:"needed"`
	Stock       quantity.Q `json:"stock"`
	Promised    quantity.Q `json:"promised"`
	Total       quantity.Q `json:"total"`
	Missing     quantity.Q `json:"missing"`
	Percent     int        `json:"percent"`
}

func (a *API) getPublicSummary(c *gin.Context) {
	ctx := c.Request.Context()
	today := time.Now()

	stockBalances, err := a.Stock.Balances(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load stock"})
		return
	}
	promisesToday, err := a.Promises.PublicAggregates(ctx, today)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load promises"})
		return
	}
	// Needs use stock + valid (non-expired) promises.
	validPromises, err := a.Promises.ValidAggregatesFromToday(ctx, today)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load promises"})
		return
	}
	comp, err := a.Basket.Get(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load basket"})
		return
	}
	targets, err := a.Settings.NeedTargets(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load settings"})
		return
	}

	promisedByFood := map[int64]quantity.Q{}
	for _, p := range validPromises {
		promisedByFood[p.FoodID] = p.Quantity
	}
	stockByFood := map[int64]stock.Balance{}
	for _, s := range stockBalances {
		stockByFood[s.FoodID] = s
	}

	var needs []needProgress
	for _, item := range comp.Items {
		needKey := "need." + item.Code
		v, ok := targets[needKey]
		if !ok {
			continue
		}
		needed, err := quantity.New(v)
		if err != nil || needed.Sign() <= 0 {
			continue
		}
		st := stockByFood[item.FoodID]
		promised := promisedByFood[item.FoodID]
		total := st.Quantity.Add(promised)
		missing := needed.Sub(total)
		if missing.Sign() < 0 {
			missing = quantity.Zero()
		}
		needs = append(needs, needProgress{
			FoodID: item.FoodID, Code: item.Code, Name: item.Name, Unit: item.Unit,
			Needed: needed, Stock: st.Quantity, Promised: promised, Total: total, Missing: missing,
			Percent: percentOf(total, needed),
		})
	}

	site, _ := a.Settings.Public(ctx)
	communityNeeds, _ := a.Needs.List(ctx, true)
	c.JSON(http.StatusOK, gin.H{
		"site":              site,
		"stock":             stockBalances,
		"available_baskets": comp.AvailableBaskets,
		"limiting_food":     gin.H{"code": comp.LimitingFoodCode, "name": comp.LimitingFoodName},
		"promises_today":    promisesToday,
		"needs":             needs,
		"community_needs":   communityNeeds,
		"as_of":             today.Format("2006-01-02"),
	})
}

func (a *API) getPublicNeeds(c *gin.Context) {
	items, err := a.Needs.List(c.Request.Context(), true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load needs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"needs": items})
}

func percentOf(total, needed quantity.Q) int {
	if needed.IsZero() {
		return 0
	}
	// (total * 100) / needed using only quantity.Q arithmetic
	acc := quantity.Zero()
	for i := 0; i < 100; i++ {
		acc = acc.Add(total)
	}
	n := acc.DivFloorInt(needed)
	if n > 100 {
		return 100
	}
	if n < 0 {
		return 0
	}
	return int(n)
}

func (a *API) getPublicStock(c *gin.Context) {
	balances, err := a.Stock.Balances(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load stock"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"stock": balances})
}

func (a *API) getPublicPromises(c *gin.Context) {
	agg, err := a.Promises.PublicAggregates(c.Request.Context(), time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load promises"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"date":     time.Now().Format("2006-01-02"),
		"promises": agg,
	})
}

// createPublicPromise lets a donor register a contribution promise without admin login.
// The response never echoes the person name back in list endpoints; only a confirmation id.
func (a *API) createPublicPromise(c *gin.Context) {
	var in promise.CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	p, err := a.Promises.Create(c.Request.Context(), in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":           p.ID,
		"food_id":      p.FoodID,
		"quantity":     p.Quantity,
		"promise_date": p.PromiseDate,
	})
}

func (a *API) getPublicSettings(c *gin.Context) {
	s, err := a.Settings.Public(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load settings"})
		return
	}
	c.JSON(http.StatusOK, s)
}

func (a *API) postLogin(c *gin.Context) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	u, err := a.Auth.Login(c, body.Username, body.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": u})
}

func (a *API) postLogout(c *gin.Context) {
	a.Auth.Logout(c)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *API) getMe(c *gin.Context) {
	u, err := a.Auth.Me(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": u})
}

func (a *API) listFoods(c *gin.Context) {
	items, err := a.Foods.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list foods"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"foods": items})
}

func (a *API) createFood(c *gin.Context) {
	var in food.CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	f, err := a.Foods.Create(c.Request.Context(), in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, f)
}

func (a *API) stockIn(c *gin.Context) {
	a.stockMove(c, true)
}

func (a *API) stockOut(c *gin.Context) {
	a.stockMove(c, false)
}

func (a *API) stockMove(c *gin.Context, isIn bool) {
	var body struct {
		FoodID   int64      `json:"food_id"`
		Quantity quantity.Q `json:"quantity"`
		Note     *string    `json:"note"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	u, _ := auth.CurrentUser(c)
	uid := u.ID
	var (
		m   stock.Movement
		err error
	)
	if isIn {
		m, err = a.Stock.MoveIn(c.Request.Context(), body.FoodID, body.Quantity, body.Note, &uid)
	} else {
		m, err = a.Stock.MoveOut(c.Request.Context(), body.FoodID, body.Quantity, body.Note, &uid)
	}
	if err != nil {
		if errors.Is(err, stock.ErrInsufficientStock) {
			c.JSON(http.StatusConflict, gin.H{"error": "insufficient stock"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, m)
}

func (a *API) listMovements(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	items, err := a.Stock.ListMovements(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list movements"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"movements": items})
}

func (a *API) listPromises(c *gin.Context) {
	var from *time.Time
	if c.Query("from") != "all" {
		t := time.Now()
		day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
		from = &day
	}
	items, err := a.Promises.ListAdmin(c.Request.Context(), from)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list promises"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"promises": items})
}

func (a *API) createPromise(c *gin.Context) {
	var in promise.CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	p, err := a.Promises.Create(c.Request.Context(), in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (a *API) deletePromise(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := a.Promises.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, promise.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *API) getBasket(c *gin.Context) {
	comp, err := a.Basket.Get(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load basket"})
		return
	}
	c.JSON(http.StatusOK, comp)
}

func (a *API) putBasket(c *gin.Context) {
	var body struct {
		Items []basket.PutItem `json:"items"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	comp, err := a.Basket.Replace(c.Request.Context(), body.Items)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, comp)
}
