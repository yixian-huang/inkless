package app

import (
	"gorm.io/gorm"

	"github.com/yixian-huang/inkless/backend/internal/repository"
)

// repos groups data-access dependencies constructed once at process start.
type repos struct {
	user                repository.UserRepository
	refreshToken        repository.RefreshTokenRepository
	contentDoc          repository.ContentDocumentRepository
	media               repository.MediaRepository
	pageView            repository.PageViewRepository
	category            repository.CategoryRepository
	tag                 repository.TagRepository
	article             repository.ArticleRepository
	articleVersion      repository.ArticleVersionRepository
	auditEvent          repository.AuditEventRepository
	page                repository.PageRepository
	installedTheme      repository.InstalledThemeRepository
	menu                repository.MenuRepository
	role                repository.RoleRepository
	marketplace         repository.MarketplaceRepository
	mediaFolder         repository.MediaFolderRepository
	chunkedUpload       repository.ChunkedUploadRepository
	glossary            repository.GlossaryRepository
	storageConfig       repository.StorageConfigRepository
	unifiedPage         repository.UnifiedPageRepository
	pageVersion         repository.PageVersionRepository
	scheduledPublishJob repository.ScheduledPublishJobRepository
	pageTemplate        repository.PageTemplateRepository
	siteConfig          repository.SiteConfigRepository
}

func wireRepos(gdb *gorm.DB) *repos {
	return &repos{
		user:                repository.NewGormUserRepository(gdb),
		refreshToken:        repository.NewGormRefreshTokenRepository(gdb),
		contentDoc:          repository.NewGormContentDocumentRepository(gdb),
		media:               repository.NewGormMediaRepository(gdb),
		pageView:            repository.NewGormPageViewRepository(gdb),
		category:            repository.NewGormCategoryRepository(gdb),
		tag:                 repository.NewGormTagRepository(gdb),
		article:             repository.NewGormArticleRepository(gdb),
		articleVersion:      repository.NewGormArticleVersionRepository(gdb),
		auditEvent:          repository.NewGormAuditEventRepository(gdb),
		page:                repository.NewGormPageRepository(gdb),
		installedTheme:      repository.NewGormInstalledThemeRepository(gdb),
		menu:                repository.NewGormMenuRepository(gdb),
		role:                repository.NewGormRoleRepository(gdb),
		marketplace:         repository.NewGormMarketplaceRepository(gdb),
		mediaFolder:         repository.NewGormMediaFolderRepository(gdb),
		chunkedUpload:       repository.NewGormChunkedUploadRepository(gdb),
		glossary:            repository.NewGormGlossaryRepository(gdb),
		storageConfig:       repository.NewGormStorageConfigRepository(gdb),
		unifiedPage:         repository.NewGormUnifiedPageRepository(gdb),
		pageVersion:         repository.NewGormPageVersionRepository(gdb),
		scheduledPublishJob: repository.NewGormScheduledPublishJobRepository(gdb),
		pageTemplate:        repository.NewGormPageTemplateRepository(gdb),
		siteConfig:          repository.NewGormSiteConfigRepository(gdb),
	}
}
