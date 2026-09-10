package errors

import "fmt"

var (
	RootAgentCreateFailed = fmt.Errorf("failed to create root agent")
	SkillCreationFailed   = fmt.Errorf("failed to create Skill")
	SkillDeletionFailed   = fmt.Errorf("failed to delete Skill")
)
