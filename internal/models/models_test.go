package models_test

import (
	"testing"

	"github.com/memohai/memoh/internal/models"
	"github.com/stretchr/testify/assert"
)

// This is an example test file demonstrating how to use the models service
// Actual tests would require database setup and mocking

func ExampleService_Create() {
	// Example usage - in real code, you would initialize with actual database connection
	// service := models.NewService(queries)

	// ctx := context.Background()
	// req := models.AddRequest{
	// 	ModelID:    "gpt-4",
	// 	Name:       "GPT-4",
	// 	LlmProviderID: "11111111-1111-1111-1111-111111111111",
	// 	Type:       models.ModelTypeChat,
	// }

	// resp, err := service.Create(ctx, req)
	// if err != nil {
	// 	// handle error
	// }
	// fmt.Printf("Created model with ID: %s\n", resp.ID)
}

func ExampleService_GetByModelID() {
	// Example usage
	// service := models.NewService(queries)

	// ctx := context.Background()
	// resp, err := service.GetByModelID(ctx, "gpt-4")
	// if err != nil {
	// 	// handle error
	// }
	// fmt.Printf("Model: %+v\n", resp.Model)
}

func ExampleService_List() {
	// Example usage
	// service := models.NewService(queries)

	// ctx := context.Background()
	// models, err := service.List(ctx)
	// if err != nil {
	// 	// handle error
	// }
	// for _, model := range models {
	// 	fmt.Printf("Model ID: %s, Type: %s\n", model.ModelID, model.Type)
	// }
}

func ExampleService_ListByType() {
	// Example usage
	// service := models.NewService(queries)

	// ctx := context.Background()
	// chatModels, err := service.ListByType(ctx, models.ModelTypeChat)
	// if err != nil {
	// 	// handle error
	// }
	// fmt.Printf("Found %d chat models\n", len(chatModels))
}

func ExampleService_UpdateByModelID() {
	// Example usage
	// service := models.NewService(queries)

	// ctx := context.Background()
	// req := models.UpdateRequest{
	// 	ModelID:    "gpt-4",
	// 	Name:       "GPT-4 Turbo",
	// 	LlmProviderID: "11111111-1111-1111-1111-111111111111",
	// 	Type:       models.ModelTypeChat,
	// }

	// resp, err := service.UpdateByModelID(ctx, "gpt-4", req)
	// if err != nil {
	// 	// handle error
	// }
	// fmt.Printf("Updated model: %s\n", resp.ModelId)
}

func ExampleService_DeleteByModelID() {
	// Example usage
	// service := models.NewService(queries)

	// ctx := context.Background()
	// err := service.DeleteByModelID(ctx, "gpt-4")
	// if err != nil {
	// 	// handle error
	// }
	// fmt.Println("Model deleted successfully")
}

func TestModel_Validate(t *testing.T) {
	tests := []struct {
		name    string
		model   models.Model
		wantErr bool
	}{
		{
			name: "valid chat model",
			model: models.Model{
				ModelID:       "gpt-4",
				Name:          "GPT-4",
				LlmProviderID: "11111111-1111-1111-1111-111111111111",
				Type:          models.ModelTypeChat,
			},
			wantErr: false,
		},
		{
			name: "valid embedding model",
			model: models.Model{
				ModelID:       "text-embedding-ada-002",
				Name:          "Ada Embeddings",
				LlmProviderID: "11111111-1111-1111-1111-111111111111",
				Type:          models.ModelTypeEmbedding,
				Dimensions:    1536,
			},
			wantErr: false,
		},
		{
			name: "missing model_id",
			model: models.Model{
				LlmProviderID: "11111111-1111-1111-1111-111111111111",
				Type:          models.ModelTypeChat,
			},
			wantErr: true,
		},
		{
			name: "missing llm_provider_id",
			model: models.Model{
				ModelID: "gpt-4",
				Type:    models.ModelTypeChat,
			},
			wantErr: true,
		},
		{
			name: "invalid llm_provider_id",
			model: models.Model{
				ModelID:       "gpt-4",
				LlmProviderID: "not-a-uuid",
				Type:          models.ModelTypeChat,
			},
			wantErr: true,
		},
		{
			name: "invalid model type",
			model: models.Model{
				ModelID:       "gpt-4",
				LlmProviderID: "11111111-1111-1111-1111-111111111111",
				Type:          "invalid",
			},
			wantErr: true,
		},
		{
			name: "embedding model missing dimensions",
			model: models.Model{
				ModelID:       "text-embedding-ada-002",
				LlmProviderID: "11111111-1111-1111-1111-111111111111",
				Type:          models.ModelTypeEmbedding,
				Dimensions:    0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.model.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestModelTypes(t *testing.T) {
	t.Run("ModelType constants", func(t *testing.T) {
		assert.Equal(t, models.ModelType("chat"), models.ModelTypeChat)
		assert.Equal(t, models.ModelType("embedding"), models.ModelTypeEmbedding)
	})

	t.Run("ClientType constants", func(t *testing.T) {
		assert.Equal(t, models.ClientType("openai"), models.ClientTypeOpenAI)
		assert.Equal(t, models.ClientType("openai-compat"), models.ClientTypeOpenAICompat)
		assert.Equal(t, models.ClientType("anthropic"), models.ClientTypeAnthropic)
		assert.Equal(t, models.ClientType("google"), models.ClientTypeGoogle)
		assert.Equal(t, models.ClientType("azure"), models.ClientTypeAzure)
		assert.Equal(t, models.ClientType("bedrock"), models.ClientTypeBedrock)
		assert.Equal(t, models.ClientType("mistral"), models.ClientTypeMistral)
		assert.Equal(t, models.ClientType("xai"), models.ClientTypeXAI)
		assert.Equal(t, models.ClientType("ollama"), models.ClientTypeOllama)
		assert.Equal(t, models.ClientType("dashscope"), models.ClientTypeDashscope)
	})
}

// Integration test example (requires actual database)
// func TestService_Integration(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("Skipping integration test")
// 	}
//
// 	ctx := context.Background()
//
// 	// Setup database connection
// 	pool, err := db.Open(ctx, config.PostgresConfig{
// 		Host:     "localhost",
// 		Port:     5432,
// 		User:     "test",
// 		Password: "test",
// 		Database: "test_db",
// 		SSLMode:  "disable",
// 	})
// 	require.NoError(t, err)
// 	defer pool.Close()
//
// 	queries := sqlc.New(pool)
// 	service := models.NewService(queries)
//
// 	// Test Create
// 	createReq := models.AddRequest{
// 		ModelID:    "test-gpt-4",
// 		Name:       "Test GPT-4",
// 		BaseURL:    "https://api.openai.com/v1",
// 		APIKey:     "sk-test",
// 		ClientType: models.ClientTypeOpenAI,
// 		Type:       models.ModelTypeChat,
// 	}
// 	createResp, err := service.Create(ctx, createReq)
// 	require.NoError(t, err)
// 	assert.NotEmpty(t, createResp.ID)
// 	assert.Equal(t, "test-gpt-4", createResp.ModelID)
//
// 	// Test GetByModelID
// 	getResp, err := service.GetByModelID(ctx, "test-gpt-4")
// 	require.NoError(t, err)
// 	assert.Equal(t, "test-gpt-4", getResp.ModelID)
// 	assert.Equal(t, "Test GPT-4", getResp.Name)
//
// 	// Test List
// 	models, err := service.List(ctx)
// 	require.NoError(t, err)
// 	assert.NotEmpty(t, models)
//
// 	// Test Update
// 	updateReq := models.UpdateRequest{
// 		ModelID:    "test-gpt-4",
// 		Name:       "Updated GPT-4",
// 		BaseURL:    "https://api.openai.com/v1",
// 		APIKey:     "sk-test-updated",
// 		ClientType: models.ClientTypeOpenAI,
// 		Type:       models.ModelTypeChat,
// 	}
// 	updateResp, err := service.UpdateByModelID(ctx, "test-gpt-4", updateReq)
// 	require.NoError(t, err)
// 	assert.Equal(t, "Updated GPT-4", updateResp.Name)
//
// 	// Test Count
// 	count, err := service.Count(ctx)
// 	require.NoError(t, err)
// 	assert.Greater(t, count, int64(0))
//
// 	// Test Delete
// 	err = service.DeleteByModelID(ctx, "test-gpt-4")
// 	require.NoError(t, err)
// }
