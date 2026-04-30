package server_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/2bit-software/zombiekit/internal/logging"
	"github.com/2bit-software/zombiekit/internal/server"
	artifactv1 "github.com/2bit-software/zombiekit/proto/gen/zombiekit/brains/artifact/v1"
	"github.com/2bit-software/zombiekit/proto/gen/zombiekit/brains/artifact/v1/artifactv1connect"
	configv1 "github.com/2bit-software/zombiekit/proto/gen/zombiekit/brains/config/v1"
	"github.com/2bit-software/zombiekit/proto/gen/zombiekit/brains/config/v1/configv1connect"
	profilev1 "github.com/2bit-software/zombiekit/proto/gen/zombiekit/brains/profile/v1"
	"github.com/2bit-software/zombiekit/proto/gen/zombiekit/brains/profile/v1/profilev1connect"
	searchv1 "github.com/2bit-software/zombiekit/proto/gen/zombiekit/brains/search/v1"
	"github.com/2bit-software/zombiekit/proto/gen/zombiekit/brains/search/v1/searchv1connect"
	workflowv1 "github.com/2bit-software/zombiekit/proto/gen/zombiekit/brains/workflow/v1"
	"github.com/2bit-software/zombiekit/proto/gen/zombiekit/brains/workflow/v1/workflowv1connect"
)

type testHarness struct {
	baseURL   string
	cancel    context.CancelFunc
	profiles  profilev1connect.ProfileServiceClient
	workflows workflowv1connect.WorkflowServiceClient
	artifacts artifactv1connect.ArtifactServiceClient
	configs   configv1connect.ConfigServiceClient
	search    searchv1connect.SearchServiceClient
}

func setupTestHarness(t *testing.T) *testHarness {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	logging.InitLogger("error", false, nil)
	t.Cleanup(logging.ResetLogger)

	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"pgvector/pgvector:pg16",
		tcpostgres.WithDatabase("test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	port := freePort(t)
	addr := fmt.Sprintf(":%d", port)

	cfg := &server.Config{
		ListenAddr:    addr,
		PostgresURL:   connStr,
		RunMigrations: true,
	}

	srv, err := server.New(cfg)
	require.NoError(t, err)

	serverCtx, cancel := context.WithCancel(ctx)
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.Run(serverCtx)
	}()

	baseURL := fmt.Sprintf("http://localhost:%d", port)

	require.Eventually(t, func() bool {
		resp, err := http.Get(baseURL + "/healthz")
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 30*time.Second, 100*time.Millisecond, "server did not become ready")

	t.Cleanup(func() {
		cancel()
		select {
		case err := <-serverErr:
			if err != nil {
				t.Logf("server exited with error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Log("server did not shut down within 5s")
		}
	})

	httpClient := http.DefaultClient

	return &testHarness{
		baseURL:   baseURL,
		cancel:    cancel,
		profiles:  profilev1connect.NewProfileServiceClient(httpClient, baseURL),
		workflows: workflowv1connect.NewWorkflowServiceClient(httpClient, baseURL),
		artifacts: artifactv1connect.NewArtifactServiceClient(httpClient, baseURL),
		configs:   configv1connect.NewConfigServiceClient(httpClient, baseURL),
		search:    searchv1connect.NewSearchServiceClient(httpClient, baseURL),
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port
}

func TestZKServer(t *testing.T) {
	h := setupTestHarness(t)
	ctx := context.Background()

	t.Run("Health", func(t *testing.T) {
		resp, err := http.Get(h.baseURL + "/healthz")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("ProfileService", func(t *testing.T) {
		t.Run("SaveAndGet", func(t *testing.T) {
			_, err := h.profiles.SaveProfile(ctx, connect.NewRequest(&profilev1.SaveProfileRequest{
				Name:     "test-profile",
				Content:  "You are a helpful assistant.",
				Location: profilev1.ProfileLocation_PROFILE_LOCATION_GLOBAL,
			}))
			require.NoError(t, err)

			resp, err := h.profiles.GetProfile(ctx, connect.NewRequest(&profilev1.GetProfileRequest{
				Name: "test-profile",
			}))
			require.NoError(t, err)
			assert.Equal(t, "test-profile", resp.Msg.Profile.Name)
			assert.Equal(t, "You are a helpful assistant.", resp.Msg.Profile.Content)
		})

		t.Run("OverwriteProtection", func(t *testing.T) {
			_, err := h.profiles.SaveProfile(ctx, connect.NewRequest(&profilev1.SaveProfileRequest{
				Name:    "test-profile",
				Content: "new content",
			}))
			require.Error(t, err)
			assert.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))
		})

		t.Run("OverwriteAllowed", func(t *testing.T) {
			resp, err := h.profiles.SaveProfile(ctx, connect.NewRequest(&profilev1.SaveProfileRequest{
				Name:      "test-profile",
				Content:   "updated content",
				Overwrite: true,
			}))
			require.NoError(t, err)
			assert.Equal(t, "updated content", resp.Msg.Profile.Content)
		})

		t.Run("List", func(t *testing.T) {
			_, err := h.profiles.SaveProfile(ctx, connect.NewRequest(&profilev1.SaveProfileRequest{
				Name:    "another-profile",
				Content: "second profile",
			}))
			require.NoError(t, err)

			resp, err := h.profiles.ListProfiles(ctx, connect.NewRequest(&profilev1.ListProfilesRequest{}))
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(resp.Msg.Profiles), 2)
		})

		t.Run("GetNotFound", func(t *testing.T) {
			_, err := h.profiles.GetProfile(ctx, connect.NewRequest(&profilev1.GetProfileRequest{
				Name: "nonexistent",
			}))
			require.Error(t, err)
			assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
		})

		t.Run("ComposeProfile", func(t *testing.T) {
			resp, err := h.profiles.ComposeProfile(ctx, connect.NewRequest(&profilev1.ComposeProfileRequest{
				ProfileNames: []string{"test-profile", "another-profile"},
			}))
			require.NoError(t, err)
			assert.Contains(t, resp.Msg.ComposedContent, "updated content")
			assert.Contains(t, resp.Msg.ComposedContent, "second profile")
			assert.Len(t, resp.Msg.ResolvedProfiles, 2)
		})

		t.Run("SaveValidation", func(t *testing.T) {
			_, err := h.profiles.SaveProfile(ctx, connect.NewRequest(&profilev1.SaveProfileRequest{
				Content: "no name",
			}))
			require.Error(t, err)
			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		})
	})

	var initiativeID string

	t.Run("WorkflowService", func(t *testing.T) {
		t.Run("CreateInitiative", func(t *testing.T) {
			resp, err := h.workflows.CreateInitiative(ctx, connect.NewRequest(&workflowv1.CreateInitiativeRequest{
				Name:        "test-feature",
				Type:        workflowv1.InitiativeType_INITIATIVE_TYPE_FEATURE,
				Description: "A test feature",
			}))
			require.NoError(t, err)
			assert.NotEmpty(t, resp.Msg.Initiative.Id)
			assert.Equal(t, "test-feature", resp.Msg.Initiative.Name)
			assert.Equal(t, workflowv1.InitiativeStatus_INITIATIVE_STATUS_IN_PROGRESS, resp.Msg.Initiative.Status)
			assert.Len(t, resp.Msg.Initiative.Steps, 4)
			initiativeID = resp.Msg.Initiative.Id
		})

		t.Run("GetStatus", func(t *testing.T) {
			resp, err := h.workflows.GetStatus(ctx, connect.NewRequest(&workflowv1.GetStatusRequest{
				InitiativeId: initiativeID,
			}))
			require.NoError(t, err)
			assert.Equal(t, initiativeID, resp.Msg.Initiative.Id)
			assert.Equal(t, "A test feature", resp.Msg.Initiative.Description)
		})

		t.Run("UpdateStep", func(t *testing.T) {
			resp, err := h.workflows.UpdateStep(ctx, connect.NewRequest(&workflowv1.UpdateStepRequest{
				InitiativeId: initiativeID,
				StepName:     "spec",
				Status:       workflowv1.StepStatus_STEP_STATUS_COMPLETED,
			}))
			require.NoError(t, err)

			for _, step := range resp.Msg.Initiative.Steps {
				if step.Name == "spec" {
					assert.Equal(t, workflowv1.StepStatus_STEP_STATUS_COMPLETED, step.Status)
				}
			}
		})

		t.Run("UpdateStepNotFound", func(t *testing.T) {
			_, err := h.workflows.UpdateStep(ctx, connect.NewRequest(&workflowv1.UpdateStepRequest{
				InitiativeId: initiativeID,
				StepName:     "nonexistent-step",
				Status:       workflowv1.StepStatus_STEP_STATUS_COMPLETED,
			}))
			require.Error(t, err)
			assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
		})

		t.Run("ListInitiatives", func(t *testing.T) {
			resp, err := h.workflows.ListInitiatives(ctx, connect.NewRequest(&workflowv1.ListInitiativesRequest{}))
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(resp.Msg.Initiatives), 1)
		})

		t.Run("ListWithStatusFilter", func(t *testing.T) {
			resp, err := h.workflows.ListInitiatives(ctx, connect.NewRequest(&workflowv1.ListInitiativesRequest{
				StatusFilter: workflowv1.InitiativeStatus_INITIATIVE_STATUS_IN_PROGRESS,
			}))
			require.NoError(t, err)
			for _, init := range resp.Msg.Initiatives {
				assert.Equal(t, workflowv1.InitiativeStatus_INITIATIVE_STATUS_IN_PROGRESS, init.Status)
			}
		})

		t.Run("CompleteInitiative", func(t *testing.T) {
			resp, err := h.workflows.CompleteInitiative(ctx, connect.NewRequest(&workflowv1.CompleteInitiativeRequest{
				InitiativeId: initiativeID,
			}))
			require.NoError(t, err)
			assert.Equal(t, workflowv1.InitiativeStatus_INITIATIVE_STATUS_COMPLETED, resp.Msg.Initiative.Status)
		})

		t.Run("GetNotFound", func(t *testing.T) {
			_, err := h.workflows.GetStatus(ctx, connect.NewRequest(&workflowv1.GetStatusRequest{
				InitiativeId: "00000000-0000-0000-0000-000000000000",
			}))
			require.Error(t, err)
			assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
		})

		t.Run("CreateValidation", func(t *testing.T) {
			_, err := h.workflows.CreateInitiative(ctx, connect.NewRequest(&workflowv1.CreateInitiativeRequest{}))
			require.Error(t, err)
			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		})
	})

	t.Run("ArtifactService", func(t *testing.T) {
		// Create a fresh initiative for artifact tests
		initResp, err := h.workflows.CreateInitiative(ctx, connect.NewRequest(&workflowv1.CreateInitiativeRequest{
			Name: "artifact-test-init",
			Type: workflowv1.InitiativeType_INITIATIVE_TYPE_FEATURE,
		}))
		require.NoError(t, err)
		artInitID := initResp.Msg.Initiative.Id

		t.Run("SaveAndGet", func(t *testing.T) {
			_, err := h.artifacts.SaveArtifact(ctx, connect.NewRequest(&artifactv1.SaveArtifactRequest{
				InitiativeId: artInitID,
				Path:         "spec/business-spec.md",
				Content:      []byte("# Business Spec\n\nThis is a test spec."),
				ContentType:  "text/markdown",
			}))
			require.NoError(t, err)

			resp, err := h.artifacts.GetArtifact(ctx, connect.NewRequest(&artifactv1.GetArtifactRequest{
				InitiativeId: artInitID,
				Path:         "spec/business-spec.md",
			}))
			require.NoError(t, err)
			assert.Equal(t, "spec/business-spec.md", resp.Msg.Artifact.Path)
			assert.Equal(t, "# Business Spec\n\nThis is a test spec.", string(resp.Msg.Artifact.Content))
			assert.Equal(t, "text/markdown", resp.Msg.Artifact.ContentType)
			assert.Equal(t, int64(len("# Business Spec\n\nThis is a test spec.")), resp.Msg.Artifact.SizeBytes)
		})

		t.Run("SaveDefaultContentType", func(t *testing.T) {
			_, err := h.artifacts.SaveArtifact(ctx, connect.NewRequest(&artifactv1.SaveArtifactRequest{
				InitiativeId: artInitID,
				Path:         "notes.txt",
				Content:      []byte("plain text"),
			}))
			require.NoError(t, err)

			resp, err := h.artifacts.GetArtifact(ctx, connect.NewRequest(&artifactv1.GetArtifactRequest{
				InitiativeId: artInitID,
				Path:         "notes.txt",
			}))
			require.NoError(t, err)
			assert.Equal(t, "text/plain", resp.Msg.Artifact.ContentType)
		})

		t.Run("List", func(t *testing.T) {
			resp, err := h.artifacts.ListArtifacts(ctx, connect.NewRequest(&artifactv1.ListArtifactsRequest{
				InitiativeId: artInitID,
			}))
			require.NoError(t, err)
			assert.Len(t, resp.Msg.Artifacts, 2)
		})

		t.Run("ListWithPrefix", func(t *testing.T) {
			resp, err := h.artifacts.ListArtifacts(ctx, connect.NewRequest(&artifactv1.ListArtifactsRequest{
				InitiativeId: artInitID,
				PathPrefix:   "spec/",
			}))
			require.NoError(t, err)
			assert.Len(t, resp.Msg.Artifacts, 1)
			assert.Equal(t, "spec/business-spec.md", resp.Msg.Artifacts[0].Path)
		})

		t.Run("GetNotFound", func(t *testing.T) {
			_, err := h.artifacts.GetArtifact(ctx, connect.NewRequest(&artifactv1.GetArtifactRequest{
				InitiativeId: artInitID,
				Path:         "nonexistent.md",
			}))
			require.Error(t, err)
			assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
		})

		t.Run("SaveValidation", func(t *testing.T) {
			_, err := h.artifacts.SaveArtifact(ctx, connect.NewRequest(&artifactv1.SaveArtifactRequest{
				Path:    "missing-initiative-id.md",
				Content: []byte("content"),
			}))
			require.Error(t, err)
			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		})

		t.Run("Overwrite", func(t *testing.T) {
			_, err := h.artifacts.SaveArtifact(ctx, connect.NewRequest(&artifactv1.SaveArtifactRequest{
				InitiativeId: artInitID,
				Path:         "spec/business-spec.md",
				Content:      []byte("# Updated Spec"),
				ContentType:  "text/markdown",
			}))
			require.NoError(t, err)

			resp, err := h.artifacts.GetArtifact(ctx, connect.NewRequest(&artifactv1.GetArtifactRequest{
				InitiativeId: artInitID,
				Path:         "spec/business-spec.md",
			}))
			require.NoError(t, err)
			assert.Equal(t, "# Updated Spec", string(resp.Msg.Artifact.Content))
		})
	})

	t.Run("ConfigService", func(t *testing.T) {
		t.Run("SetAndGet", func(t *testing.T) {
			_, err := h.configs.UpdateConfig(ctx, connect.NewRequest(&configv1.UpdateConfigRequest{
				Entries: []*configv1.Config{
					{Key: "theme", Value: "dark"},
					{Key: "lang", Value: "en"},
				},
			}))
			require.NoError(t, err)

			resp, err := h.configs.GetConfig(ctx, connect.NewRequest(&configv1.GetConfigRequest{
				Keys: []string{"theme"},
			}))
			require.NoError(t, err)
			require.Len(t, resp.Msg.Entries, 1)
			assert.Equal(t, "theme", resp.Msg.Entries[0].Key)
			assert.Equal(t, "dark", resp.Msg.Entries[0].Value)
		})

		t.Run("GetAll", func(t *testing.T) {
			resp, err := h.configs.GetConfig(ctx, connect.NewRequest(&configv1.GetConfigRequest{}))
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(resp.Msg.Entries), 2)
		})

		t.Run("UpdateExisting", func(t *testing.T) {
			_, err := h.configs.UpdateConfig(ctx, connect.NewRequest(&configv1.UpdateConfigRequest{
				Entries: []*configv1.Config{
					{Key: "theme", Value: "light"},
				},
			}))
			require.NoError(t, err)

			resp, err := h.configs.GetConfig(ctx, connect.NewRequest(&configv1.GetConfigRequest{
				Keys: []string{"theme"},
			}))
			require.NoError(t, err)
			require.Len(t, resp.Msg.Entries, 1)
			assert.Equal(t, "light", resp.Msg.Entries[0].Value)
		})
	})

	t.Run("SearchService", func(t *testing.T) {
		t.Run("SearchUnavailable", func(t *testing.T) {
			_, err := h.search.Search(ctx, connect.NewRequest(&searchv1.SearchRequest{
				Query: "test query",
			}))
			require.Error(t, err)
			assert.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
		})

		t.Run("ListConversationsEmpty", func(t *testing.T) {
			resp, err := h.search.ListConversations(ctx, connect.NewRequest(&searchv1.ListConversationsRequest{}))
			require.NoError(t, err)
			assert.Empty(t, resp.Msg.Conversations)
		})
	})
}
