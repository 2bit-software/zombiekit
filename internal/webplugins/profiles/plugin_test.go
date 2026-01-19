package profiles_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zombiekit/brains/internal/profile"
	"github.com/zombiekit/brains/internal/search"
	"github.com/zombiekit/brains/internal/webplugins/profiles"
)

// mockSource implements profile.ProfileSourceInterface for testing.
type mockSource struct {
	profiles map[string]*profile.Profile
}

func (m *mockSource) FindProfileDirs() ([]profile.ResolvedDirectory, error) {
	return []profile.ResolvedDirectory{
		{Path: "/mock", Source: profile.SourceLocal},
	}, nil
}

func (m *mockSource) LoadProfiles(dirs []profile.ResolvedDirectory) (map[string]*profile.Profile, error) {
	return m.profiles, nil
}

func (m *mockSource) LoadAllProfiles(dirs []profile.ResolvedDirectory) (map[string][]*profile.Profile, error) {
	result := make(map[string][]*profile.Profile)
	for name, p := range m.profiles {
		result[name] = []*profile.Profile{p}
	}
	return result, nil
}

func (m *mockSource) GetInheritanceChain(name string) ([]*profile.Profile, error) {
	if p, ok := m.profiles[name]; ok {
		return []*profile.Profile{p}, nil
	}
	return nil, nil
}

func (m *mockSource) CreateProfile(name string, global bool) (string, error) {
	return "/mock/" + name + ".md", nil
}

func (m *mockSource) GetInitDir(global bool) (string, error) {
	return "/mock", nil
}

func (m *mockSource) SourceName() string {
	return "mock"
}

func (m *mockSource) DefaultInherits() bool {
	return true
}

func newTestPlugin(profs map[string]*profile.Profile) *profiles.Plugin {
	source := &mockSource{profiles: profs}
	service := profile.NewServiceWithSourceInterface(source)
	return profiles.NewPlugin(service)
}

// === User Story 4 Tests: Search across names and content ===

func TestSearchMatchesName(t *testing.T) {
	plugin := newTestPlugin(map[string]*profile.Profile{
		"developer": {Name: "developer", Description: "A coding expert", Source: profile.SourceLocal},
		"designer":  {Name: "designer", Description: "A design specialist", Source: profile.SourceLocal},
		"devops":    {Name: "devops", Description: "Infrastructure expert", Source: profile.SourceLocal},
	})

	results, err := plugin.Search("dev", 10, search.SortRelevance)
	require.NoError(t, err)
	assert.Len(t, results, 2, "should match 'developer' and 'devops'")

	// Verify titles
	titles := make([]string, len(results))
	for i, r := range results {
		titles[i] = r.Title
	}
	assert.Contains(t, titles, "developer")
	assert.Contains(t, titles, "devops")
}

func TestSearchMatchesContent(t *testing.T) {
	plugin := newTestPlugin(map[string]*profile.Profile{
		"frontend":  {Name: "frontend", Description: "React and Vue specialist", Source: profile.SourceLocal},
		"backend":   {Name: "backend", Description: "Node.js developer", Source: profile.SourceLocal},
		"fullstack": {Name: "fullstack", Description: "Python and Django expert", Source: profile.SourceLocal},
	})

	// Search for something only in description
	results, err := plugin.Search("react", 10, search.SortRelevance)
	require.NoError(t, err)
	assert.Len(t, results, 1, "should match 'frontend' by description")
	assert.Equal(t, "frontend", results[0].Title)
}

func TestSearchNoDuplicates(t *testing.T) {
	plugin := newTestPlugin(map[string]*profile.Profile{
		"developer": {Name: "developer", Description: "developer tools specialist", Source: profile.SourceLocal},
	})

	// "developer" appears in both name and description
	results, err := plugin.Search("developer", 10, search.SortRelevance)
	require.NoError(t, err)
	assert.Len(t, results, 1, "should not have duplicates when matching both name and content")
	assert.Equal(t, "developer", results[0].Title)
}

func TestSearchCaseInsensitive(t *testing.T) {
	plugin := newTestPlugin(map[string]*profile.Profile{
		"Developer": {Name: "Developer", Description: "A coding EXPERT", Source: profile.SourceLocal},
	})

	testCases := []string{"developer", "DEVELOPER", "Developer", "dEvElOpEr"}
	for _, query := range testCases {
		results, err := plugin.Search(query, 10, search.SortRelevance)
		require.NoError(t, err, "query: %s", query)
		assert.Len(t, results, 1, "should find result for query: %s", query)
	}

	// Also test case-insensitive description search
	results, err := plugin.Search("expert", 10, search.SortRelevance)
	require.NoError(t, err)
	assert.Len(t, results, 1, "should find 'EXPERT' with lowercase query")
}

// === User Story 2 Tests: Search results sorted by relevance ===

func TestSearchDefaultSortIsRelevance(t *testing.T) {
	plugin := newTestPlugin(map[string]*profile.Profile{
		"dev":       {Name: "dev", Description: "Short name", Source: profile.SourceLocal},
		"developer": {Name: "developer", Description: "Contains dev", Source: profile.SourceLocal},
		"devops":    {Name: "devops", Description: "Also starts with dev", Source: profile.SourceLocal},
	})

	// Search with empty sort order (should default to relevance)
	results, err := plugin.Search("dev", 10, "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)

	// Exact match should be first
	assert.Equal(t, "dev", results[0].Title, "exact match should be first")
}

func TestSearchExactMatchBeforePartial(t *testing.T) {
	plugin := newTestPlugin(map[string]*profile.Profile{
		"developer": {Name: "developer", Description: "Full name", Source: profile.SourceLocal},
		"dev":       {Name: "dev", Description: "Short name", Source: profile.SourceLocal},
		"devtools":  {Name: "devtools", Description: "Prefix match", Source: profile.SourceLocal},
	})

	results, err := plugin.Search("dev", 10, search.SortRelevance)
	require.NoError(t, err)
	require.Len(t, results, 3)

	// Exact match should be first
	assert.Equal(t, "dev", results[0].Title, "exact match should come first")
}

func TestSearchEmptySortOrderDefaultsToRelevance(t *testing.T) {
	plugin := newTestPlugin(map[string]*profile.Profile{
		"test":      {Name: "test", Description: "Exact match", Source: profile.SourceLocal},
		"testing":   {Name: "testing", Description: "Prefix match", Source: profile.SourceLocal},
		"mytest":    {Name: "mytest", Description: "Contains match", Source: profile.SourceLocal},
		"unrelated": {Name: "unrelated", Description: "testing in desc", Source: profile.SourceLocal},
	})

	// Empty string should work like SortRelevance
	results, err := plugin.Search("test", 10, "")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(results), 1)

	// Exact match "test" should be first
	assert.Equal(t, "test", results[0].Title)
}

// === User Story 3 Tests: Search results sorted by user-specified order ===

func TestSearchSortByName(t *testing.T) {
	plugin := newTestPlugin(map[string]*profile.Profile{
		"zebra":  {Name: "zebra", Description: "developer profile", Source: profile.SourceLocal},
		"alpha":  {Name: "alpha", Description: "developer profile", Source: profile.SourceLocal},
		"middle": {Name: "middle", Description: "developer profile", Source: profile.SourceLocal},
	})

	results, err := plugin.Search("developer", 10, search.SortName)
	require.NoError(t, err)
	require.Len(t, results, 3)

	// Should be sorted A-Z
	assert.Equal(t, "alpha", results[0].Title)
	assert.Equal(t, "middle", results[1].Title)
	assert.Equal(t, "zebra", results[2].Title)
}

func TestSearchSortByCreatedDate(t *testing.T) {
	// Profiles don't have timestamps, so this should fall back to relevance
	plugin := newTestPlugin(map[string]*profile.Profile{
		"dev":    {Name: "dev", Description: "Exact match", Source: profile.SourceLocal},
		"devops": {Name: "devops", Description: "Prefix match", Source: profile.SourceLocal},
	})

	results, err := plugin.Search("dev", 10, search.SortCreatedDate)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Should fall back to relevance (exact match first)
	assert.Equal(t, "dev", results[0].Title)
}

func TestSearchSortByUpdatedDate(t *testing.T) {
	// Profiles don't have timestamps, so this should fall back to relevance
	plugin := newTestPlugin(map[string]*profile.Profile{
		"dev":    {Name: "dev", Description: "Exact match", Source: profile.SourceLocal},
		"devops": {Name: "devops", Description: "Prefix match", Source: profile.SourceLocal},
	})

	results, err := plugin.Search("dev", 10, search.SortUpdatedDate)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Should fall back to relevance (exact match first)
	assert.Equal(t, "dev", results[0].Title)
}

func TestSearchSortByLastUsed(t *testing.T) {
	// Profiles don't track last used, should fall back to relevance
	plugin := newTestPlugin(map[string]*profile.Profile{
		"dev":    {Name: "dev", Description: "Exact match", Source: profile.SourceLocal},
		"devops": {Name: "devops", Description: "Prefix match", Source: profile.SourceLocal},
	})

	results, err := plugin.Search("dev", 10, search.SortLastUsed)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Should fall back to relevance (exact match first)
	assert.Equal(t, "dev", results[0].Title)
}

func TestSearchInvalidSortOrderFallsBackToRelevance(t *testing.T) {
	plugin := newTestPlugin(map[string]*profile.Profile{
		"dev":    {Name: "dev", Description: "Exact match", Source: profile.SourceLocal},
		"devops": {Name: "devops", Description: "Prefix match", Source: profile.SourceLocal},
	})

	results, err := plugin.Search("dev", 10, "invalid_sort_order")
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Should fall back to relevance (exact match first)
	assert.Equal(t, "dev", results[0].Title)
}

// === Additional edge case tests ===

func TestSearchEmptyQuery(t *testing.T) {
	plugin := newTestPlugin(map[string]*profile.Profile{
		"developer": {Name: "developer", Description: "A developer", Source: profile.SourceLocal},
	})

	results, err := plugin.Search("", 10, search.SortRelevance)
	require.NoError(t, err)
	assert.Empty(t, results, "empty query should return empty results")
}

func TestSearchNoMatches(t *testing.T) {
	plugin := newTestPlugin(map[string]*profile.Profile{
		"developer": {Name: "developer", Description: "A developer", Source: profile.SourceLocal},
	})

	results, err := plugin.Search("nonexistent", 10, search.SortRelevance)
	require.NoError(t, err)
	assert.Empty(t, results, "no matches should return empty results")
	assert.NotNil(t, results, "results should never be nil")
}

func TestSearchMaxResults(t *testing.T) {
	plugin := newTestPlugin(map[string]*profile.Profile{
		"dev1": {Name: "dev1", Description: "Developer 1", Source: profile.SourceLocal},
		"dev2": {Name: "dev2", Description: "Developer 2", Source: profile.SourceLocal},
		"dev3": {Name: "dev3", Description: "Developer 3", Source: profile.SourceLocal},
		"dev4": {Name: "dev4", Description: "Developer 4", Source: profile.SourceLocal},
		"dev5": {Name: "dev5", Description: "Developer 5", Source: profile.SourceLocal},
	})

	results, err := plugin.Search("dev", 3, search.SortRelevance)
	require.NoError(t, err)
	assert.Len(t, results, 3, "should respect maxResults limit")
}

func TestSearchURLFormat(t *testing.T) {
	plugin := newTestPlugin(map[string]*profile.Profile{
		"my-profile": {Name: "my-profile", Description: "A test profile", Source: profile.SourceLocal},
	})

	results, err := plugin.Search("profile", 10, search.SortRelevance)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "/my-profile", results[0].URL, "URL should be relative - system prefixes automatically")
}
