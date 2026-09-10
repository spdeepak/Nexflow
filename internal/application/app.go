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
	deviceID     uuid.UUID
	validator    *validator.Validate
}

func NewApp(agentService agents.Service, kbService skills.Service, modelService modelcredentials.Service, userService users.Service, chatService *runner.Chat, mcpService mcpserver.Service, deviceID uuid.UUID) *App {
	return &App{
		agentService: agentService,
		skillService: kbService,
		modelService: modelService,
		deviceID:     deviceID,
		userService:  userService,
		chatService:  chatService,
		mcpService:   mcpService,
		validator:    validator.New(),
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GetUsername() (string, error) {
	appUser, err := a.userService.GetUserByExternalID(a.ctx, a.deviceID.String())
	slog.DebugContext(a.ctx, "user", "appUser", appUser)
	if err == nil {
		return appUser.Name, nil
	}
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}
	createAppUser, err := a.userService.CreateAppUser(a.ctx, a.deviceID, currentUser.Name)
	if err != nil {
		return "", err
	}
	return createAppUser.Name, nil
}

func (a *App) Validate(obj any) error {
	err := a.validator.Struct(obj)
	if err != nil {
		var errors strings.Builder
		errors.WriteString("Invalid field values: ")
		for _, ve := range err.(validator.ValidationErrors) {
			ve.Field()
			errors.WriteString(ve.StructField())
			errors.WriteString(", ")
		}
		return fmt.Errorf("%s", errors.String())
	}
	return nil
}
