package router

import (
	"log/slog"

	"github.com/agridispatch/agridispatch/internal/config"
	"github.com/agridispatch/agridispatch/internal/constants"
	"github.com/agridispatch/agridispatch/internal/handler"
	"github.com/agridispatch/agridispatch/internal/middleware"
	"github.com/agridispatch/agridispatch/internal/service"
	"github.com/agridispatch/agridispatch/internal/ws"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// Setup 组装路由。
func Setup(
	db *gorm.DB,
	redisClient *redis.Client,
	authSvc *service.AuthService,
	dashboardSvc *service.DashboardService,
	dispatchSvc *service.DispatchService,
	hub *ws.Hub,
	cfg *config.Config,
	logger *slog.Logger,
) *gin.Engine {
	r := gin.New()
	r.Use(middleware.RequestID(), middleware.RequestLog(logger), middleware.Recovery(logger), middleware.CORS(cfg.CORSOrigins()))

	healthHandler := handler.NewHealthHandler(db, redisClient)
	authHandler := handler.NewAuthHandler(authSvc)
	dashboardHandler := handler.NewDashboardHandler(dashboardSvc)
	dispatchHandler := handler.NewDispatchHandler(dispatchSvc)

	r.GET("/healthz", healthHandler.Healthz)
	r.GET("/readyz", healthHandler.Readyz)

	v1 := r.Group("/api/" + constants.APIVersion)
	v1.GET("/healthz", healthHandler.Healthz)
	v1.GET("/readyz", healthHandler.Readyz)

	auth := v1.Group("/auth")
	{
		auth.POST("/login", middleware.RateLimit(cfg.AuthRateLimit, cfg.AuthRateWindowSecs), authHandler.Login)
		auth.GET("/me", middleware.Auth(authSvc), authHandler.Me)
	}

	dash := v1.Group("/dashboard")
	{
		dash.GET("/overview", dashboardHandler.Overview)
		dash.GET("/reports/work/export", dashboardHandler.ExportReport)
	}

	// 任务派单确认与改派（前端调用 /api/tasks/...，经 Nginx 映射到 /api/v1/tasks/...）
	v1.POST("/tasks/:id/dispatch", dispatchHandler.Confirm)
	v1.POST("/tasks/:id/reassign", dispatchHandler.Reassign)

	// WebSocket 实时轨迹
	r.GET("/ws", func(c *gin.Context) {
		hub.ServeWS(c.Writer, c.Request)
	})

	r.GET("/swagger/doc.json", healthHandler.SwaggerJSON)
	r.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger/doc.json")))
	return r
}
