package runner

import (
	"context"
	"io"

	"google.golang.org/adk/v2/tool/skilltoolset/skill"
)

type InMemorySkill struct {
	Name         string
	Description  string
	Instructions string
}

type StringSource struct {
	skills map[string]InMemorySkill
}

func NewStringToolSet(skills ...InMemorySkill) *StringSource {
	m := make(map[string]InMemorySkill, len(skills))
	for _, s := range skills {
		m[s.Name] = s
	}
	return &StringSource{skills: m}
}

func (s *StringSource) ListFrontmatters(ctx context.Context) ([]*skill.Frontmatter, error) {
	var out []*skill.Frontmatter
	for _, sk := range s.skills {
		out = append(out, &skill.Frontmatter{Name: sk.Name, Description: sk.Description})
	}
	return out, nil
}

func (s *StringSource) LoadFrontmatter(ctx context.Context, name string) (*skill.Frontmatter, error) {
	sk, ok := s.skills[name]
	if !ok {
		return nil, skill.ErrSkillNotFound
	}
	return &skill.Frontmatter{Name: sk.Name, Description: sk.Description}, nil
}

func (s *StringSource) LoadInstructions(ctx context.Context, name string) (string, error) {
	sk, ok := s.skills[name]
	if !ok {
		return "", skill.ErrSkillNotFound
	}
	return sk.Instructions, nil
}

func (s *StringSource) ListResources(ctx context.Context, name, subpath string) ([]string, error) {
	return nil, nil
}

func (s *StringSource) LoadResource(ctx context.Context, name, resourcePath string) (io.ReadCloser, error) {
	return nil, skill.ErrResourceNotFound
}
