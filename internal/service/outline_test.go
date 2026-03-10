package service

import (
	"context"
	"errors"
	"testing"

	"go-sigil/internal/models"
)

func TestOutline_ForFile(t *testing.T) {
	ctx := context.Background()

	syms := []models.Symbol{
		{ID: "aabb0001", Kind: "function", Name: "Init", Signature: "func Init()", Depth: 0},
		{ID: "aabb0002", Kind: "method", Name: "Run", Signature: "func (s *Server) Run()", ParentID: "aabb0001", Depth: 1},
	}

	tests := []struct {
		name     string
		file     string
		storeFn  func(ctx context.Context, file string) ([]models.Symbol, error)
		wantLen  int
		wantErr  bool
	}{
		{
			name: "returns symbols for file",
			file: "internal/server.go",
			storeFn: func(_ context.Context, file string) ([]models.Symbol, error) {
				if file == "internal/server.go" {
					return syms, nil
				}
				return nil, nil
			},
			wantLen: 2,
		},
		{
			name: "empty file has no symbols",
			file: "internal/empty.go",
			storeFn: func(_ context.Context, _ string) ([]models.Symbol, error) {
				return nil, nil
			},
			wantLen: 0,
		},
		{
			name: "store error propagates",
			file: "internal/bad.go",
			storeFn: func(_ context.Context, _ string) ([]models.Symbol, error) {
				return nil, errors.New("db error")
			},
			wantErr: true,
		},
		{
			name: "symbol fields mapped correctly",
			file: "internal/server.go",
			storeFn: func(_ context.Context, _ string) ([]models.Symbol, error) {
				return syms, nil
			},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &mockStore{getSymbolsByFileFn: tt.storeFn}
			o := NewOutline(st)

			result, err := o.ForFile(ctx, tt.file)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ForFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if result.File != tt.file {
				t.Errorf("File = %q, want %q", result.File, tt.file)
			}
			if len(result.Symbols) != tt.wantLen {
				t.Errorf("len(Symbols) = %d, want %d", len(result.Symbols), tt.wantLen)
			}
			if tt.wantLen > 0 {
				first := result.Symbols[0]
				if first.ID != syms[0].ID {
					t.Errorf("first.ID = %q, want %q", first.ID, syms[0].ID)
				}
				if first.Kind != syms[0].Kind {
					t.Errorf("first.Kind = %q, want %q", first.Kind, syms[0].Kind)
				}
				if first.Signature != syms[0].Signature {
					t.Errorf("first.Signature = %q, want %q", first.Signature, syms[0].Signature)
				}
			}
		})
	}
}
