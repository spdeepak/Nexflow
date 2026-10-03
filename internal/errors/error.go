package errors

import "fmt"

var (
	RootAgentCreateFailed = fmt.Errorf("failed to create root agent")
	SkillCreationFailed   = fmt.Errorf("failed to create Skill")
	SkillDeletionFailed   = fmt.Errorf("failed to delete Skill")
	SkillInvalidLocation  = fmt.Errorf("invalid skill location: the selected folder must contain a SKILL.md file")
)
