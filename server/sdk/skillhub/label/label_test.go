package label_test

import (
	"context"
	"testing"

	"miqro-skillhub/server/sdk/skillhub/label"
)

// mockLabelDefinitionRepo
type mockLabelDefRepo struct {
	defs   map[int64]label.LabelDefinition
	nextID int64
}

func newMockLabelDefRepo() *mockLabelDefRepo {
	return &mockLabelDefRepo{defs: make(map[int64]label.LabelDefinition), nextID: 1}
}
func (m *mockLabelDefRepo) FindByID(_ context.Context, id int64) (*label.LabelDefinition, error) {
	d, ok := m.defs[id]
	if !ok {
		return nil, nil
	}
	return &d, nil
}
func (m *mockLabelDefRepo) FindBySlug(_ context.Context, slug string) (*label.LabelDefinition, error) {
	for _, d := range m.defs {
		if d.Slug == slug {
			return &d, nil
		}
	}
	return nil, nil
}
func (m *mockLabelDefRepo) FindAll(_ context.Context) ([]label.LabelDefinition, error) {
	var all []label.LabelDefinition
	for _, d := range m.defs {
		all = append(all, d)
	}
	return all, nil
}
func (m *mockLabelDefRepo) FindVisible(_ context.Context) ([]label.LabelDefinition, error) {
	var visible []label.LabelDefinition
	for _, d := range m.defs {
		if d.VisibleInFilter {
			visible = append(visible, d)
		}
	}
	return visible, nil
}
func (m *mockLabelDefRepo) FindByIDs(_ context.Context, ids []int64) ([]label.LabelDefinition, error) {
	var defs []label.LabelDefinition
	for _, id := range ids {
		if d, ok := m.defs[id]; ok {
			defs = append(defs, d)
		}
	}
	return defs, nil
}
func (m *mockLabelDefRepo) Count(_ context.Context) (int64, error) {
	return int64(len(m.defs)), nil
}
func (m *mockLabelDefRepo) Save(_ context.Context, d label.LabelDefinition) (label.LabelDefinition, error) {
	if d.ID == 0 {
		d.ID = m.nextID
		m.nextID++
	}
	m.defs[d.ID] = d
	return d, nil
}
func (m *mockLabelDefRepo) Delete(_ context.Context, id int64) error {
	delete(m.defs, id)
	return nil
}

// mockLabelTranslationRepo
type mockLabelTransRepo struct {
	trans []label.LabelTranslation
}

func newMockLabelTransRepo() *mockLabelTransRepo { return &mockLabelTransRepo{} }
func (m *mockLabelTransRepo) FindByLabelID(_ context.Context, labelID int64) ([]label.LabelTranslation, error) {
	var out []label.LabelTranslation
	for _, t := range m.trans {
		if t.LabelID == labelID {
			out = append(out, t)
		}
	}
	return out, nil
}
func (m *mockLabelTransRepo) FindByLabelIDs(_ context.Context, labelIDs []int64) ([]label.LabelTranslation, error) {
	idSet := make(map[int64]bool, len(labelIDs))
	for _, id := range labelIDs {
		idSet[id] = true
	}
	var out []label.LabelTranslation
	for _, t := range m.trans {
		if idSet[t.LabelID] {
			out = append(out, t)
		}
	}
	return out, nil
}
func (m *mockLabelTransRepo) SaveAll(_ context.Context, translations []label.LabelTranslation) ([]label.LabelTranslation, error) {
	m.trans = append(m.trans, translations...)
	return translations, nil
}
func (m *mockLabelTransRepo) DeleteAll(_ context.Context, translations []label.LabelTranslation) error {
	var remaining []label.LabelTranslation
	for _, existing := range m.trans {
		keep := true
		for _, toDelete := range translations {
			if existing.LabelID == toDelete.LabelID && existing.Locale == toDelete.Locale {
				keep = false
				break
			}
		}
		if keep {
			remaining = append(remaining, existing)
		}
	}
	m.trans = remaining
	return nil
}
func (m *mockLabelTransRepo) DeleteByLabelID(_ context.Context, labelID int64) error {
	var remaining []label.LabelTranslation
	for _, t := range m.trans {
		if t.LabelID != labelID {
			remaining = append(remaining, t)
		}
	}
	m.trans = remaining
	return nil
}

// mockSkillLabelRepo
type mockSkillLabelRepo struct {
	labels map[int64]label.SkillLabel
	nextID int64
}

func newMockSkillLabelRepo() *mockSkillLabelRepo {
	return &mockSkillLabelRepo{labels: make(map[int64]label.SkillLabel), nextID: 1}
}
func (m *mockSkillLabelRepo) FindBySkillID(_ context.Context, skillID int64) ([]label.SkillLabel, error) {
	var out []label.SkillLabel
	for _, sl := range m.labels {
		if sl.SkillID == skillID {
			out = append(out, sl)
		}
	}
	return out, nil
}
func (m *mockSkillLabelRepo) FindBySkillIDs(_ context.Context, skillIDs []int64) ([]label.SkillLabel, error) {
	idSet := make(map[int64]bool, len(skillIDs))
	for _, id := range skillIDs {
		idSet[id] = true
	}
	var out []label.SkillLabel
	for _, sl := range m.labels {
		if idSet[sl.SkillID] {
			out = append(out, sl)
		}
	}
	return out, nil
}
func (m *mockSkillLabelRepo) FindByLabelID(_ context.Context, labelID int64) ([]label.SkillLabel, error) {
	var out []label.SkillLabel
	for _, sl := range m.labels {
		if sl.LabelID == labelID {
			out = append(out, sl)
		}
	}
	return out, nil
}
func (m *mockSkillLabelRepo) FindBySkillIDAndLabelID(_ context.Context, skillID int64, labelID int64) (*label.SkillLabel, error) {
	for _, sl := range m.labels {
		if sl.SkillID == skillID && sl.LabelID == labelID {
			return &sl, nil
		}
	}
	return nil, nil
}
func (m *mockSkillLabelRepo) CountBySkillID(_ context.Context, skillID int64) (int64, error) {
	var count int64
	for _, sl := range m.labels {
		if sl.SkillID == skillID {
			count++
		}
	}
	return count, nil
}
func (m *mockSkillLabelRepo) Save(_ context.Context, sl label.SkillLabel) (label.SkillLabel, error) {
	if sl.ID == 0 {
		sl.ID = m.nextID
		m.nextID++
	}
	m.labels[sl.ID] = sl
	return sl, nil
}
func (m *mockSkillLabelRepo) Delete(_ context.Context, id int64) error {
	delete(m.labels, id)
	return nil
}

func TestLabel_ValidateSlug(t *testing.T) {
	tests := []struct {
		slug    string
		wantErr bool
	}{
		{"hello-world", false},
		{"foo", false},
		{"a-b-c", false},
		{"", true},
		{"Hello-World", false}, // normalized to lowercase
		{"invalid_slug", true},
		{"-leading", true},
		{"trailing-", true},
	}
	for _, tt := range tests {
		err := label.ValidateSlug(tt.slug)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateSlug(%q) error=%v wantErr=%v", tt.slug, err, tt.wantErr)
		}
	}
}

func TestLabel_CreateDefinition(t *testing.T) {
	defRepo := newMockLabelDefRepo()
	transRepo := newMockLabelTransRepo()
	svc := label.NewLabelDefinitionService(defRepo, transRepo, nil)

	def, err := svc.Create(context.Background(), "ai-assistant", label.TypeRecommended, true, 1,
		"admin-1", map[string]bool{"LABEL_ADMIN": true},
		[]label.LabelTranslation{
			{Locale: "en", DisplayName: "AI Assistant"},
			{Locale: "zh", DisplayName: "AI助手"},
		})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if def.Slug != "ai-assistant" {
		t.Errorf("expected slug ai-assistant, got %s", def.Slug)
	}

	// Verify translations.
	trans, _ := svc.GetTranslations(context.Background(), def.ID)
	if len(trans) != 2 {
		t.Fatalf("expected 2 translations, got %d", len(trans))
	}
}

func TestLabel_Create_NoPermission(t *testing.T) {
	defRepo := newMockLabelDefRepo()
	transRepo := newMockLabelTransRepo()
	svc := label.NewLabelDefinitionService(defRepo, transRepo, nil)

	_, err := svc.Create(context.Background(), "test", label.TypeRecommended, true, 1,
		"user-1", nil, nil)
	if err == nil {
		t.Fatal("expected noPermission error")
	}
}

func TestLabel_AssignSkillLabel(t *testing.T) {
	skillLabelRepo := newMockSkillLabelRepo()
	svc := label.NewSkillLabelService(skillLabelRepo, nil, nil)

	sl, err := svc.Assign(context.Background(), 100, 1, "admin-1", map[string]bool{"LABEL_ADMIN": true})
	if err != nil {
		t.Fatalf("Assign failed: %v", err)
	}
	if sl.SkillID != 100 {
		t.Errorf("expected skillID 100, got %d", sl.SkillID)
	}

	got, _ := svc.GetForSkill(context.Background(), 100)
	if len(got) != 1 {
		t.Fatalf("expected 1 label on skill, got %d", len(got))
	}
}

func TestLabel_AssignSkillLabel_NoPermission(t *testing.T) {
	skillLabelRepo := newMockSkillLabelRepo()
	svc := label.NewSkillLabelService(skillLabelRepo, nil, nil)

	_, err := svc.Assign(context.Background(), 100, 1, "user-1", nil)
	if err == nil {
		t.Fatal("expected noPermission error")
	}
}
