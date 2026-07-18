package module

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"blotting-consultancy/internal/cache"
	"blotting-consultancy/internal/provider"
	"blotting-consultancy/internal/repository"
)

// Module defines the contract for a self-contained feature module.
type Module interface {
	Name() string
	Init(deps Dependencies) error
	RegisterRoutes(public, admin *gin.RouterGroup)
}

// Dependencies provides shared resources that modules need.
type Dependencies struct {
	DB         *gorm.DB
	Registry   *provider.Registry
	Repos      *SharedRepos
	SiteCfg    repository.SiteConfigRepository
	UserRepo   repository.UserRepository // for RBAC middleware in modules
	RBACCache  *cache.Cache              // for RBAC middleware in modules
	UploadDir  string                    // path to uploads directory
	BackupDir  string                    // path to database backup archives
	AppVersion string                    // application version string
}

// SharedRepos holds cross-module repositories.
type SharedRepos struct {
	ContentDoc repository.ContentDocumentRepository
	Article    repository.ArticleRepository
}
