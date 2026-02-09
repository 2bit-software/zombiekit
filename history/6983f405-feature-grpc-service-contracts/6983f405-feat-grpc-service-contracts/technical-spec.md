# Technical Specification: gRPC Service Contracts

## Proto File Templates

### Common Patterns

All proto files follow these patterns:

```protobuf
syntax = "proto3";

package zombiekit.brains.<service>.v1;

import "google/protobuf/timestamp.proto";
import "zombiekit/brains/common/v1/common.proto";

// Request messages include request_id for traceability
message FooRequest {
  string request_id = 1;
  // ... other fields
}

// Entity messages reserve fields 10-19 for future multi-tenant use
message Entity {
  string id = 1;
  // ... fields 2-9
  reserved 10 to 19;
  reserved "tenant_id", "organization_id";
}
```

---

## Service Definitions

### 1. common.proto

```protobuf
syntax = "proto3";

package zombiekit.brains.common.v1;

import "google/protobuf/timestamp.proto";

// Pagination request fields (embed in list requests)
message PageRequest {
  int32 page_size = 1;
  string page_token = 2;
}

// Pagination response fields (embed in list responses)
message PageResponse {
  string next_page_token = 1;
  int32 total_count = 2;
}

// Audit issue from spec/code audits
message AuditIssue {
  Severity severity = 1;
  string message = 2;
  string file_path = 3;
  int32 line_number = 4;
  string suggestion = 5;

  enum Severity {
    SEVERITY_UNSPECIFIED = 0;
    SEVERITY_CRITICAL = 1;
    SEVERITY_MAJOR = 2;
    SEVERITY_MINOR = 3;
  }
}

// Complexity analysis result
message ComplexityResult {
  int32 score = 1;
  repeated ComplexityFactor factors = 2;
  string recommendation = 3;
}

message ComplexityFactor {
  string name = 1;
  int32 value = 2;
  float weight = 3;
}
```

---

### 2. workflow.proto (WorkflowService)

```protobuf
syntax = "proto3";

package zombiekit.brains.workflow.v1;

import "google/protobuf/timestamp.proto";
import "zombiekit/brains/common/v1/common.proto";

service WorkflowService {
  rpc CreateInitiative(CreateInitiativeRequest) returns (CreateInitiativeResponse);
  rpc GetStatus(GetStatusRequest) returns (GetStatusResponse);
  rpc UpdateStep(UpdateStepRequest) returns (UpdateStepResponse);
  rpc CompleteInitiative(CompleteInitiativeRequest) returns (CompleteInitiativeResponse);
  rpc ListInitiatives(ListInitiativesRequest) returns (ListInitiativesResponse);
}

// Enums
enum InitiativeType {
  INITIATIVE_TYPE_UNSPECIFIED = 0;
  INITIATIVE_TYPE_FEATURE = 1;
  INITIATIVE_TYPE_BUG = 2;
  INITIATIVE_TYPE_REFACTOR = 3;
}

enum InitiativeStatus {
  INITIATIVE_STATUS_UNSPECIFIED = 0;
  INITIATIVE_STATUS_IN_PROGRESS = 1;
  INITIATIVE_STATUS_COMPLETED = 2;
}

enum StepStatus {
  STEP_STATUS_UNSPECIFIED = 0;
  STEP_STATUS_PENDING = 1;
  STEP_STATUS_IN_PROGRESS = 2;
  STEP_STATUS_COMPLETED = 3;
  STEP_STATUS_SKIPPED = 4;
}

// Messages
message Initiative {
  string id = 1;
  string name = 2;
  InitiativeType type = 3;
  InitiativeStatus status = 4;
  repeated WorkflowStep steps = 5;
  google.protobuf.Timestamp created_at = 6;
  google.protobuf.Timestamp updated_at = 7;
  string description = 8;
  string branch_name = 9;

  reserved 10 to 19;
  reserved "tenant_id", "organization_id";
}

message WorkflowStep {
  string name = 1;
  StepStatus status = 2;
  google.protobuf.Timestamp updated_at = 3;

  reserved 10 to 19;
  reserved "tenant_id", "organization_id";
}

// RPCs
message CreateInitiativeRequest {
  string request_id = 1;
  string name = 2;
  InitiativeType type = 3;
  string description = 4;
}

message CreateInitiativeResponse {
  Initiative initiative = 1;
}

message GetStatusRequest {
  string request_id = 1;
  string initiative_id = 2;  // If empty, returns active initiative
}

message GetStatusResponse {
  Initiative initiative = 1;
}

message UpdateStepRequest {
  string request_id = 1;
  string initiative_id = 2;
  string step_name = 3;
  StepStatus status = 4;
}

message UpdateStepResponse {
  Initiative initiative = 1;
}

message CompleteInitiativeRequest {
  string request_id = 1;
  string initiative_id = 2;
}

message CompleteInitiativeResponse {
  Initiative initiative = 1;
}

message ListInitiativesRequest {
  string request_id = 1;
  zombiekit.brains.common.v1.PageRequest pagination = 2;
  InitiativeStatus status_filter = 3;
}

message ListInitiativesResponse {
  repeated Initiative initiatives = 1;
  zombiekit.brains.common.v1.PageResponse pagination = 2;
}
```

---

### 3. profile.proto (ProfileService)

```protobuf
syntax = "proto3";

package zombiekit.brains.profile.v1;

import "google/protobuf/timestamp.proto";
import "zombiekit/brains/common/v1/common.proto";

service ProfileService {
  // MVP RPCs
  rpc ComposeProfile(ComposeProfileRequest) returns (ComposeProfileResponse);
  rpc ListProfiles(ListProfilesRequest) returns (ListProfilesResponse);
  rpc GetProfile(GetProfileRequest) returns (GetProfileResponse);
  rpc SaveProfile(SaveProfileRequest) returns (SaveProfileResponse);

  // Followup: Streaming (contract defined, implementation deferred)
  rpc SubscribeProfileUpdates(SubscribeProfileUpdatesRequest) returns (stream ProfileUpdateEvent);
}

// Enums
enum ProfileLocation {
  PROFILE_LOCATION_UNSPECIFIED = 0;
  PROFILE_LOCATION_LOCAL = 1;
  PROFILE_LOCATION_GLOBAL = 2;
}

enum ProfileEventType {
  PROFILE_EVENT_TYPE_UNSPECIFIED = 0;
  PROFILE_EVENT_TYPE_CREATED = 1;
  PROFILE_EVENT_TYPE_UPDATED = 2;
  PROFILE_EVENT_TYPE_DELETED = 3;
}

// Messages
message Profile {
  string name = 1;
  string content = 2;
  repeated string domains = 3;
  repeated string dependencies = 4;
  ProfileLocation location = 5;
  google.protobuf.Timestamp created_at = 6;
  google.protobuf.Timestamp updated_at = 7;

  reserved 10 to 19;
  reserved "tenant_id", "organization_id";
}

// MVP RPCs
message ComposeProfileRequest {
  string request_id = 1;
  repeated string profile_names = 2;
  string working_directory = 3;
}

message ComposeProfileResponse {
  string composed_content = 1;
  repeated string resolved_profiles = 2;
}

message ListProfilesRequest {
  string request_id = 1;
  string working_directory = 2;
}

message ListProfilesResponse {
  repeated Profile profiles = 1;
}

message GetProfileRequest {
  string request_id = 1;
  string name = 2;
  string working_directory = 3;
}

message GetProfileResponse {
  Profile profile = 1;
}

message SaveProfileRequest {
  string request_id = 1;
  string name = 2;
  string content = 3;
  ProfileLocation location = 4;
  bool overwrite = 5;
  string working_directory = 6;
}

message SaveProfileResponse {
  Profile profile = 1;
}

// Streaming (Followup)
message SubscribeProfileUpdatesRequest {
  string request_id = 1;
}

message ProfileUpdateEvent {
  ProfileEventType event_type = 1;
  string profile_name = 2;
  google.protobuf.Timestamp timestamp = 3;
  Profile profile = 4;  // Included for CREATED/UPDATED events
}
```

---

### 4. search.proto (SearchService)

```protobuf
syntax = "proto3";

package zombiekit.brains.search.v1;

import "google/protobuf/timestamp.proto";
import "zombiekit/brains/common/v1/common.proto";

service SearchService {
  rpc Search(SearchRequest) returns (SearchResponse);
  rpc GetConversation(GetConversationRequest) returns (GetConversationResponse);
  rpc ListConversations(ListConversationsRequest) returns (ListConversationsResponse);
}

// Messages
message Conversation {
  string id = 1;
  string project = 2;
  google.protobuf.Timestamp created_at = 3;
  google.protobuf.Timestamp updated_at = 4;
  string summary = 5;
  int32 total_chunks = 6;

  reserved 10 to 19;
  reserved "tenant_id", "organization_id";
}

message ConversationChunk {
  string id = 1;
  string conversation_id = 2;
  string content = 3;
  google.protobuf.Timestamp created_at = 4;
  int32 sequence = 5;

  reserved 10 to 19;
  reserved "tenant_id", "organization_id";
}

message SearchResult {
  ConversationChunk chunk = 1;
  float score = 2;
}

// RPCs
message SearchRequest {
  string request_id = 1;
  string query = 2;
  int32 limit = 3;
  string project_filter = 4;
}

message SearchResponse {
  repeated SearchResult results = 1;
}

message GetConversationRequest {
  string request_id = 1;
  string conversation_id = 2;
  zombiekit.brains.common.v1.PageRequest pagination = 3;
}

message GetConversationResponse {
  Conversation conversation = 1;
  repeated ConversationChunk chunks = 2;
  zombiekit.brains.common.v1.PageResponse pagination = 3;
}

message ListConversationsRequest {
  string request_id = 1;
  zombiekit.brains.common.v1.PageRequest pagination = 2;
  string project_filter = 3;
}

message ListConversationsResponse {
  repeated Conversation conversations = 1;
  zombiekit.brains.common.v1.PageResponse pagination = 2;
}
```

---

### 5. artifact.proto (ArtifactService)

```protobuf
syntax = "proto3";

package zombiekit.brains.artifact.v1;

import "google/protobuf/timestamp.proto";

service ArtifactService {
  rpc GetArtifact(GetArtifactRequest) returns (GetArtifactResponse);
  rpc SaveArtifact(SaveArtifactRequest) returns (SaveArtifactResponse);
  rpc ListArtifacts(ListArtifactsRequest) returns (ListArtifactsResponse);
}

// Messages
message Artifact {
  string initiative_id = 1;
  string path = 2;
  bytes content = 3;  // bytes for binary compatibility
  google.protobuf.Timestamp created_at = 4;
  google.protobuf.Timestamp updated_at = 5;
  string content_type = 6;
  int64 size_bytes = 7;

  reserved 10 to 19;
  reserved "tenant_id", "organization_id";
}

// RPCs
message GetArtifactRequest {
  string request_id = 1;
  string initiative_id = 2;
  string path = 3;
}

message GetArtifactResponse {
  Artifact artifact = 1;
}

message SaveArtifactRequest {
  string request_id = 1;
  string initiative_id = 2;
  string path = 3;
  bytes content = 4;
  string content_type = 5;
}

message SaveArtifactResponse {
  Artifact artifact = 1;
}

message ListArtifactsRequest {
  string request_id = 1;
  string initiative_id = 2;
  string path_prefix = 3;
}

message ListArtifactsResponse {
  repeated Artifact artifacts = 1;
}
```

---

### 6. config.proto (ConfigService)

```protobuf
syntax = "proto3";

package zombiekit.brains.config.v1;

import "google/protobuf/timestamp.proto";

service ConfigService {
  // MVP RPCs
  rpc GetConfig(GetConfigRequest) returns (GetConfigResponse);
  rpc UpdateConfig(UpdateConfigRequest) returns (UpdateConfigResponse);

  // Followup: Streaming (contract defined, implementation deferred)
  rpc SubscribeConfigUpdates(SubscribeConfigUpdatesRequest) returns (stream ConfigUpdateEvent);
}

// Enums
enum ConfigEventType {
  CONFIG_EVENT_TYPE_UNSPECIFIED = 0;
  CONFIG_EVENT_TYPE_UPDATED = 1;
}

// Messages
message Config {
  string key = 1;
  string value = 2;
  google.protobuf.Timestamp updated_at = 3;

  reserved 10 to 19;
  reserved "tenant_id", "organization_id";
}

// RPCs
message GetConfigRequest {
  string request_id = 1;
  repeated string keys = 2;  // If empty, returns all config
}

message GetConfigResponse {
  repeated Config entries = 1;
}

message UpdateConfigRequest {
  string request_id = 1;
  repeated Config entries = 2;
}

message UpdateConfigResponse {
  repeated Config entries = 1;
}

// Streaming (Followup)
message SubscribeConfigUpdatesRequest {
  string request_id = 1;
}

message ConfigUpdateEvent {
  ConfigEventType event_type = 1;
  string key = 2;
  string value = 3;
  google.protobuf.Timestamp timestamp = 4;
}
```

---

### 7. llm.proto (LLMService) - Followup

```protobuf
syntax = "proto3";

package zombiekit.brains.llm.v1;

// LLMService - Contract defined, implementation deferred to followup ticket
service LLMService {
  rpc Complete(CompletionRequest) returns (CompletionResponse);
  rpc CompleteStream(CompletionRequest) returns (stream CompletionChunk);
}

// Enums
enum Role {
  ROLE_UNSPECIFIED = 0;
  ROLE_SYSTEM = 1;
  ROLE_USER = 2;
  ROLE_ASSISTANT = 3;
}

enum FinishReason {
  FINISH_REASON_UNSPECIFIED = 0;
  FINISH_REASON_STOP = 1;
  FINISH_REASON_LENGTH = 2;
  FINISH_REASON_CONTENT_FILTER = 3;
}

// Messages
message Message {
  Role role = 1;
  string content = 2;
}

message Usage {
  int32 prompt_tokens = 1;
  int32 completion_tokens = 2;
  int32 total_tokens = 3;
}

message CompletionRequest {
  string request_id = 1;
  string model = 2;
  repeated Message messages = 3;
  int32 max_tokens = 4;
  float temperature = 5;
}

message CompletionResponse {
  string content = 1;
  FinishReason finish_reason = 2;
  Usage usage = 3;
}

message CompletionChunk {
  string delta_content = 1;
  FinishReason finish_reason = 2;  // Set on final chunk
  Usage usage = 3;  // Set on final chunk
}
```

---

## Generated Code Structure

After running `buf generate`:

```
gen/
└── zombiekit/
    └── brains/
        ├── common/
        │   └── v1/
        │       └── common.pb.go
        ├── workflow/
        │   └── v1/
        │       ├── workflow.pb.go
        │       └── workflowv1connect/
        │           └── workflow.connect.go
        ├── profile/
        │   └── v1/
        │       ├── profile.pb.go
        │       └── profilev1connect/
        │           └── profile.connect.go
        ├── search/
        │   └── v1/
        │       ├── search.pb.go
        │       └── searchv1connect/
        │           └── search.connect.go
        ├── artifact/
        │   └── v1/
        │       ├── artifact.pb.go
        │       └── artifactv1connect/
        │           └── artifact.connect.go
        ├── config/
        │   └── v1/
        │       ├── config.pb.go
        │       └── configv1connect/
        │           └── config.connect.go
        └── llm/
            └── v1/
                ├── llm.pb.go
                └── llmv1connect/
                    └── llm.connect.go
```

## Usage Example (Future Reference)

```go
// Server implementation
type workflowServer struct {}

func (s *workflowServer) CreateInitiative(
    ctx context.Context,
    req *connect.Request[workflowv1.CreateInitiativeRequest],
) (*connect.Response[workflowv1.CreateInitiativeResponse], error) {
    // Implementation here
}

// Client usage
client := workflowv1connect.NewWorkflowServiceClient(
    http.DefaultClient,
    "https://server.example.com",
)
resp, err := client.CreateInitiative(ctx, connect.NewRequest(&workflowv1.CreateInitiativeRequest{
    RequestId: uuid.New().String(),
    Name:      "my-feature",
    Type:      workflowv1.InitiativeType_INITIATIVE_TYPE_FEATURE,
}))
```
