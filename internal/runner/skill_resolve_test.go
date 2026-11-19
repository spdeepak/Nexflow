package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/spdeepak/nexflow/internal/enums"
	"github.com/spdeepak/nexflow/internal/schema"
)

// writeSkillDir lays out <tmp>/<dirName>/SKILL.md and returns the SKILL.md
// path (the shape resolveSkills expects for a storage-uri skill).
func writeSkillDir(t *testing.T, dirName string) string {
	t.Helper()
	skillDir := filepath.Join(t.TempDir(), dirName)
	require.NoError(t, os.MkdirAll(skillDir, 0o700))
	skillMD := "---\nname: " + dirName + "\ndescription: a test skill\n---\n\nDo the thing.\n"
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0o600))
	return filepath.Join(skillDir, "SKILL.md")
}

// The skills_content_check constraint allows a row with NULL content as long
// as storage_uri is set (and vice versa), so resolveSkills must branch on the
// data present rather than dereferencing Content blindly. A broken skill row
// must be skipped, never panic or fail the run.
func TestResolveSkills_SkipsUnusableSkills(t *testing.T) {
	inline := "Inline instructions."
	storageURI := writeSkillDir(t, "fs-skill")
	missingURI := filepath.Join(t.TempDir(), "does-not-exist", "SKILL.md")

	agent := schema.AgentDetail{Skills: []schema.Skill{
		// (a) normal inline skill.
		{ID: uuid.New(), Title: "inline-skill", Content: &inline, ContentType: enums.SkillContentTypeText},
		// (b) storage-uri skill whose content_type matches.
		{ID: uuid.New(), Title: "fs-skill", StorageUri: &storageURI, ContentType: enums.SkillContentTypeStorageUri},
		// (c) legal per DB constraint but previously panicked: NULL content.
		{ID: uuid.New(), Title: "nil-content", ContentType: enums.SkillContentTypeText},
		// (d) storage_uri points at a path that does not exist.
		{ID: uuid.New(), Title: "bad-path", StorageUri: &missingURI, ContentType: enums.SkillContentTypeStorageUri},
	}}

	r := &runner{}
	ts, err := r.resolveSkills(context.Background(), agent)
	require.NoError(t, err)
	require.NotNil(t, ts)

	fms, err := skillSource(context.Background(), agent.Skills).ListFrontmatters(context.Background())
	require.NoError(t, err)
	names := make([]string, 0, len(fms))
	for _, fm := range fms {
		names = append(names, fm.Name)
	}
	require.ElementsMatch(t, []string{"inline-skill", "fs-skill"}, names,
		"only skills with usable content or a valid storage path must reach the toolset")
}

// A storage-uri skill whose content_type is NOT 'storageUri' still carries a
// usable StorageUri; branching on data present routes it to the file source
// instead of dereferencing nil Content.
func TestResolveSkills_StorageUriWithoutStorageContentType(t *testing.T) {
	storageURI := writeSkillDir(t, "mislabelled-skill")
	agent := schema.AgentDetail{Skills: []schema.Skill{
		{ID: uuid.New(), Title: "mislabelled-skill", StorageUri: &storageURI, ContentType: enums.SkillContentTypeText},
	}}

	r := &runner{}
	ts, err := r.resolveSkills(context.Background(), agent)
	require.NoError(t, err)
	require.NotNil(t, ts)

	fms, err := skillSource(context.Background(), agent.Skills).ListFrontmatters(context.Background())
	require.NoError(t, err)
	require.Len(t, fms, 1)
	require.Equal(t, "mislabelled-skill", fms[0].Name)
}
