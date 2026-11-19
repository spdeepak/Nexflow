package errors

import "fmt"

var (
	ErrRootAgentCreateFailed = fmt.Errorf("failed to create root agent")
	ErrSkillCreationFailed   = fmt.Errorf("failed to create Skill")
	ErrSkillDeletionFailed   = fmt.Errorf("failed to delete Skill")
	ErrSkillInvalidLocation  = fmt.Errorf("invalid skill location: the selected folder must contain a SKILL.md file")
	ErrSkillTitleTaken       = fmt.Errorf("a skill with this title already exists")
	ErrSkillNotFound         = fmt.Errorf("skill not found or not owned by this user")
)
