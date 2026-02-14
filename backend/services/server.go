package services

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/server"
	"github.com/NicoClack/cryptic-stash/backend/server/endpoints"
	"github.com/NicoClack/cryptic-stash/backend/server/middleware"
	"github.com/NicoClack/cryptic-stash/backend/server/servercommon"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Server struct {
	App    *common.App
	Router *gin.Engine
	Server *http.Server
}

func NewServer(app *common.App) *Server {
	router := gin.New()
	if app.Env.PROXY_ORIGINAL_IP_HEADER_NAME == "" {
		_ = router.SetTrustedProxies(nil)
	} else {
		router.TrustedPlatform = app.Env.PROXY_ORIGINAL_IP_HEADER_NAME
	}
	router.Use(middleware.NewLogger(app.Logger))
	router.Use(gin.Logger()) // TODO: make the custom logger log completed requests so this isn't needed
	// TODO: ^ this is logging "Error #01: ..."
	router.Use(middleware.NewTimeout())

	if len(app.Env.ALLOWED_ORIGINS) > 0 {
		corsConfig := cors.DefaultConfig()
		corsConfig.AllowOrigins = app.Env.ALLOWED_ORIGINS
		corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
		router.Use(cors.New(corsConfig))
	}

	router.LoadHTMLFS(http.FS(server.TemplateFiles.FS), fmt.Sprintf("%v/*.html", server.TemplateFiles.Path))
	router.Use(middleware.NewRateLimiting("api", app.RateLimiter))
	router.Use(middleware.NewError())

	adminMiddleware := middleware.NewAdminProtected(app.Core)
	serverApp := &servercommon.ServerApp{
		App:             app,
		Router:          router,
		AdminMiddleware: adminMiddleware,
	}
	endpoints.ConfigureEndpoints(&servercommon.Group{
		RouterGroup: router.Group(""),
		App:         serverApp,
	})

	router.NoRoute(middleware.NewStaticFS(server.PublicFiles.FS, server.PublicFiles.Path))

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%v", app.Env.PORT),
		Handler:           router.Handler(),
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       3 * time.Second,
		WriteTimeout:      3 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	return &Server{
		App:    app,
		Router: router,
		Server: httpServer,
	}
}

func (service *Server) Start() {
	go func() {
		stdErr := service.Server.ListenAndServe()
		if stdErr != nil && !errors.Is(stdErr, http.ErrServerClosed) {
			log.Fatalf("an error occurred while starting the HTTP server:\n%v", stdErr.Error())
		}
	}()
}
func (service *Server) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stdErr := service.Server.Shutdown(ctx)
	if stdErr != nil {
		service.App.Logger.Warn("an error occurred while shutting down the HTTP server", stdErr)
	}
}

func (service *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	service.Server.Handler.ServeHTTP(w, r)
}
