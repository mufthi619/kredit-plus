package main

import (
	"context"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"go.uber.org/zap"
	"kredit-plus/config"
	loggerPkg "kredit-plus/infra/logger"
	"kredit-plus/infra/mysql"
	"kredit-plus/infra/redis"
	"kredit-plus/wire"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	ctx := context.Background()

	//Init Logger
	logger, err := loggerPkg.NewLogger(loggerPkg.Config(cfg.Logger))
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %v", err))
	}
	defer logger.Sync()

	//Init Otel
	//shutdown, err := telemetry.InitTracer(ctx, telemetry.Config(cfg.Telemetry), logger)
	//if err != nil {
	//	logger.Fatal("failed to initialize telemetry", zap.Error(err))
	//}
	//defer shutdown()

	//Init GORM (MySQL)
	db, err := mysql.NewClient(ctx, mysql.Config(cfg.MySQL), logger)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	//Init Redis
	redisClient, err := redis.NewClient(redis.Config(cfg.Redis), logger)
	if err != nil {
		logger.Fatal("failed to connect to redis", zap.Error(err))
	}
	defer redisClient.Close()

	//Server (Fiber)
	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
	})
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
	}))

	//Asset
	assetHandler, err := wire.InitializeAssetHandler(db, redisClient, logger)
	if err != nil {
		logger.Fatal("failed to initialize asset handler", zap.Error(err))
	}
	assetHandler.RegisterRoutes(app)
	//Customer
	customerHandler, err := wire.InitializeCustomerHandler(db, redisClient, logger)
	if err != nil {
		logger.Fatal("failed to initialize customer handler", zap.Error(err))
	}
	customerHandler.RegisterRoutes(app)
	//Credit Limit
	creditLimitHandler, err := wire.InitializeCreditLimitHandler(db, redisClient, logger)
	if err != nil {
		logger.Fatal("failed to initialize credit limit handler", zap.Error(err))
	}
	creditLimitHandler.RegisterRoutes(app)
	//Transaction
	transactionHandler, err := wire.InitializeTransactionProviderHandler(db, redisClient, logger)
	if err != nil {
		logger.Fatal("failed to initialize transaction handler", zap.Error(err))
	}
	transactionHandler.RegisterRoutes(app)

	//Start Server
	go func() {
		if err := app.Listen(fmt.Sprintf(":%d", cfg.App.Port)); err != nil {
			logger.Fatal("failed to start server", zap.Error(err))
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")
	if err := app.Shutdown(); err != nil {
		logger.Fatal("server forced to shutdown", zap.Error(err))
	}
}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	return c.Status(code).JSON(fiber.Map{
		"code":    code,
		"message": err.Error(),
	})
}
