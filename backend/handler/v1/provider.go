package v1

import (
	"github.com/google/wire"

	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/middleware"
	"github.com/chaitin/panda-wiki/usecase"
)

type APIHandlers struct {
	UserHandler             *UserHandler
	KnowledgeBaseHandler    *KnowledgeBaseHandler
	NodeHandler             *NodeHandler
	AppHandler              *AppHandler
	FileHandler             *FileHandler
	ModelHandler            *ModelHandler
	ConversationHandler     *ConversationHandler
	CrawlerHandler          *CrawlerHandler
	CreationHandler         *CreationHandler
	StatHandler             *StatHandler
	CommentHandler          *CommentHandler
	AuthV1Handler           *AuthV1Handler
	DocumentFeedbackHandler *DocumentFeedbackHandler
	ConsultHandler          *ConsultHandler
	StandardTermHandler     *StandardTermHandler
	TestEnvHandler          *TestEnvHandler
	CloudAuthHandler        *CloudAuthHandler
	ExpertProfileHandler    *ExpertProfileHandler
	AuditHandler            *AuditHandler
	ApprovalHandler         *ApprovalHandler
	BackupHandler           *BackupHandler
	SystemInfoHandler       *SystemInfoHandler
}

var ProviderSet = wire.NewSet(
	middleware.ProviderSet,
	usecase.ProviderSet,

	wire.Bind(new(middleware.AuditRecorder), new(*usecase.AuditUsecase)),
	middleware.NewAuditMiddleware,

	handler.NewBaseHandler,
	NewNodeHandler,
	NewAppHandler,
	NewConversationHandler,
	NewUserHandler,
	NewFileHandler,
	NewModelHandler,
	NewKnowledgeBaseHandler,
	NewCrawlerHandler,
	NewCreationHandler,
	NewStatHandler,
	NewCommentHandler,
	NewAuthV1Handler,
	NewDocumentFeedbackHandler,
	NewConsultHandler,
	NewStandardTermHandler,
	NewTestEnvHandler,
	NewCloudAuthHandler,
	NewExpertProfileHandler,
	NewAuditHandler,
	NewApprovalHandler,
	NewBackupHandler,
	NewSystemInfoHandler,

	wire.Struct(new(APIHandlers), "*"),
)
