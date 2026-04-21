package adkapp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/adk/filesystem"
	adkskill "github.com/cloudwego/eino/adk/middlewares/skill"
)

type osFilesystemBackend struct{}

func (osFilesystemBackend) LsInfo(ctx context.Context, req *filesystem.LsInfoRequest) ([]filesystem.FileInfo, error) {
	_ = ctx
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	entries, err := os.ReadDir(req.Path)
	if err != nil {
		return nil, err
	}
	out := make([]filesystem.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, statErr := entry.Info()
		if statErr != nil {
			return nil, statErr
		}
		out = append(out, toFileInfo(filepath.Join(req.Path, entry.Name()), info))
	}
	return out, nil
}

func (osFilesystemBackend) Read(ctx context.Context, req *filesystem.ReadRequest) (*filesystem.FileContent, error) {
	_ = ctx
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	data, err := os.ReadFile(req.FilePath)
	if err != nil {
		return nil, err
	}
	return &filesystem.FileContent{Content: string(data)}, nil
}

func (osFilesystemBackend) GrepRaw(ctx context.Context, req *filesystem.GrepRequest) ([]filesystem.GrepMatch, error) {
	_ = ctx
	_ = req
	return nil, fmt.Errorf("grep is not supported")
}

func (osFilesystemBackend) GlobInfo(ctx context.Context, req *filesystem.GlobInfoRequest) ([]filesystem.FileInfo, error) {
	_ = ctx
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	matches, err := filepath.Glob(filepath.Join(req.Path, req.Pattern))
	if err != nil {
		return nil, err
	}
	out := make([]filesystem.FileInfo, 0, len(matches))
	for _, match := range matches {
		info, statErr := os.Stat(match)
		if statErr != nil {
			return nil, statErr
		}
		out = append(out, toFileInfo(match, info))
	}
	return out, nil
}

func (osFilesystemBackend) Write(ctx context.Context, req *filesystem.WriteRequest) error {
	_ = ctx
	if req == nil {
		return fmt.Errorf("request is required")
	}
	return os.WriteFile(req.FilePath, []byte(req.Content), 0o644)
}

func (osFilesystemBackend) Edit(ctx context.Context, req *filesystem.EditRequest) error {
	_ = ctx
	_ = req
	return fmt.Errorf("edit is not supported")
}

type interviewOnlySkillBackend struct {
	base adkskill.Backend
}

func (b interviewOnlySkillBackend) List(ctx context.Context) ([]adkskill.FrontMatter, error) {
	items, err := b.base.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]adkskill.FrontMatter, 0, len(items))
	for _, item := range items {
		if !looksLikeInterviewSkill(item.Name, item.Description) {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

func (b interviewOnlySkillBackend) Get(ctx context.Context, name string) (adkskill.Skill, error) {
	skill, err := b.base.Get(ctx, name)
	if err != nil {
		return adkskill.Skill{}, err
	}
	if !looksLikeInterviewSkill(skill.Name, skill.Description) {
		return adkskill.Skill{}, fmt.Errorf("skill %q is not an interview skill", name)
	}
	return skill, nil
}

func looksLikeInterviewSkill(name, description string) bool {
	text := strings.ToLower(strings.TrimSpace(name + " " + description))
	return strings.Contains(text, "interview")
}

func toFileInfo(path string, info os.FileInfo) filesystem.FileInfo {
	return filesystem.FileInfo{
		Path:       path,
		IsDir:      info.IsDir(),
		Size:       info.Size(),
		ModifiedAt: info.ModTime().UTC().Format(timeLayout),
	}
}

const timeLayout = "2006-01-02T15:04:05Z07:00"
