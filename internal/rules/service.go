package rules

// Service provides rules resolution by combining the resolver and matcher.
type Service struct {
	workingDir string
	homeDir    string
}

// NewService creates a new rules Service.
func NewService(workingDir, homeDir string) *Service {
	return &Service{
		workingDir: workingDir,
		homeDir:    homeDir,
	}
}

// ResolveForFile returns all rules matching the given file path.
// Unconditional rules are excluded — use ResolveUnconditional for those.
// Rules with empty body are filtered out.
func (s *Service) ResolveForFile(filePath string) ([]*Rule, error) {
	allRules, err := s.loadAll()
	if err != nil {
		return nil, err
	}
	matched := MatchRules(allRules, filePath)
	return filterEmptyBody(matched), nil
}

// ResolveUnconditional returns all rules with no paths field.
// Rules with empty body are filtered out.
func (s *Service) ResolveUnconditional() ([]*Rule, error) {
	allRules, err := s.loadAll()
	if err != nil {
		return nil, err
	}

	var unconditional []*Rule
	for _, rule := range allRules {
		if rule.IsUnconditional() {
			unconditional = append(unconditional, rule)
		}
	}
	return filterEmptyBody(unconditional), nil
}

// ResolveForFiles returns deduplicated rules matching any of the given file paths.
// Unconditional rules are excluded. Rules with empty body are filtered out.
func (s *Service) ResolveForFiles(filePaths []string) ([]*Rule, error) {
	allRules, err := s.loadAll()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var matched []*Rule

	for _, filePath := range filePaths {
		for _, rule := range MatchRules(allRules, filePath) {
			id := rule.ID()
			if !seen[id] {
				seen[id] = true
				matched = append(matched, rule)
			}
		}
	}
	return filterEmptyBody(matched), nil
}

func (s *Service) loadAll() ([]*Rule, error) {
	resolver, err := NewResolver(s.workingDir, s.homeDir)
	if err != nil {
		return nil, err
	}

	dirs := resolver.FindRulesDirs()
	return resolver.LoadRules(dirs), nil
}

func filterEmptyBody(rules []*Rule) []*Rule {
	var filtered []*Rule
	for _, r := range rules {
		if r.Body != "" {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
