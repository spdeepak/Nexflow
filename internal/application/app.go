package application

import (
	"context"
	"fmt"
	"log/slog"
	"os/user"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/spdeepak/nexflow/internal/agents"
	"github.com/spdeepak/nexflow/internal/mcpserver"
	"github.com/spdeepak/nexflow/internal/mcpstore"
	"github.com/spdeepak/nexflow/internal/modelcredentials"
	"github.com/spdeepak/nexflow/internal/runner"
	"github.com/spdeepak/nexflow/internal/skills"
	"github.com/spdeepak/nexflow/internal/users"
)

type App struct {
	ctx          context.Context
	agentService agents.Service
	skillService skills.Service
	modelService modelcredentials.Service
	mcpService   mcpserver.Service
	userService  users.Service
	chatService  *runner.Chat
	tokenStore   mcpstore.TokenStore
	// deviceID is this machine's hardware serial number (or the persisted
	// fallback generated on first run when the serial was unavailable).
	deviceID string
	// userID is the local app user row bound to this device. The device
	// serial is the external identity; the row keeps its own UUID key.
	userID    uuid.UUID
	validator *validator.Validate
}

func NewApp(agentService agents.Service, kbService skills.Service, modelService modelcredentials.Service, userService users.Service, chatService *runner.Chat, mcpService mcpserver.Service, deviceID string, userID uuid.UUID, tokenStore mcpstore.TokenStore) *App {
	return &App{
		agentService: agentService,
		skillService: kbService,
		modelService: modelService,
		deviceID:     deviceID,
		userID:       userID,
		userService:  userService,
		chatService:  chatService,
		mcpService:   mcpService,
		tokenStore:   tokenStore,
		validator:    validator.New(),
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GetUsername() (string, error) {
	// Only consulted when the user row has to be created for the first time.
	name := ""
	if currentUser, err := user.Current(); err == nil {
		name = currentUser.Name
	}
	appUser, err := a.userService.EnsureAppUser(a.ctx, a.deviceID, name)
	if err != nil {
		return "", err
	}
	slog.DebugContext(a.ctx, "user", "appUser", appUser)
	return appUser.Name, nil
}

func (a *App) Validate(obj any) error {
	err := a.validator.Struct(obj)
	if err == nil {
		return nil
	}
	var msg strings.Builder
	msg.WriteString("Invalid field values: ")
	for _, ve := range err.(validator.ValidationErrors) {
		msg.WriteString(ve.StructField())
		msg.WriteString(", ")
	}
	return fmt.Errorf("%s", msg.String())
}
