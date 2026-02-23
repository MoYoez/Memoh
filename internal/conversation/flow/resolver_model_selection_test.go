package flow

import (
	"testing"

	"github.com/memohai/memoh/internal/models"
)

func TestMatchesModelReference_ModelID(t *testing.T) {
	t.Parallel()

	model := models.GetResponse{
		ID:      "a55f0d2d-1547-49a0-b085-ec4ab778f4b8",
		ModelID: "gpt-4o",
	}

	if !matchesModelReference(model, "gpt-4o") {
		t.Fatal("expected model slug to match")
	}
}

func TestMatchesModelReference_UUID(t *testing.T) {
	t.Parallel()

	model := models.GetResponse{
		ID:      "a55f0d2d-1547-49a0-b085-ec4ab778f4b8",
		ModelID: "gpt-4o",
	}

	if !matchesModelReference(model, "a55f0d2d-1547-49a0-b085-ec4ab778f4b8") {
		t.Fatal("expected model UUID to match")
	}
}

func TestMatchesModelReference_NoMatch(t *testing.T) {
	t.Parallel()

	model := models.GetResponse{
		ID:      "a55f0d2d-1547-49a0-b085-ec4ab778f4b8",
		ModelID: "gpt-4o",
	}

	if matchesModelReference(model, "gpt-4.1") {
		t.Fatal("expected non-matching model reference to fail")
	}
}

func TestMatchesModelReference_TrimmedInput(t *testing.T) {
	t.Parallel()

	model := models.GetResponse{
		ID:      "a55f0d2d-1547-49a0-b085-ec4ab778f4b8",
		ModelID: "gpt-4o",
	}

	if !matchesModelReference(model, "  gpt-4o  ") {
		t.Fatal("expected trimmed model slug to match")
	}
}
