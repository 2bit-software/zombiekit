package github

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"time"

	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit"
	gh "github.com/google/go-github/v84/github"
)

const (
	defaultTimeout         = 30 * time.Second
	defaultRateLimitThresh = 10
	maxRetries             = 3
	retryBaseDelay         = 1 * time.Second
	defaultMaxJitter       = 500 * time.Millisecond
	maxPreemptiveSleep     = 60 * time.Second
)

var _ Client = (*httpClient)(nil)

type httpClient struct {
	ghClient           *gh.Client
	owner              string
	repo               string
	endpoint           string
	customHTTPClient   *http.Client
	retryBase          time.Duration
	retryMaxJitter     time.Duration
	rateLimitThreshold int
}

func NewClient(token, owner, repo string, opts ...Option) (*httpClient, error) {
	if token == "" {
		return nil, fmt.Errorf("github: token must not be empty")
	}
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("github: owner and repo must not be empty")
	}

	c := &httpClient{
		owner:              owner,
		repo:               repo,
		retryBase:          retryBaseDelay,
		retryMaxJitter:     defaultMaxJitter,
		rateLimitThreshold: defaultRateLimitThresh,
	}
	for _, opt := range opts {
		opt(c)
	}

	var ghClient *gh.Client
	if c.customHTTPClient != nil {
		ghClient = gh.NewClient(c.customHTTPClient).WithAuthToken(token)
	} else {
		rateLimiter := github_ratelimit.NewClient(nil)
		ghClient = gh.NewClient(rateLimiter).WithAuthToken(token)
	}

	if c.endpoint != "" {
		var err error
		ghClient, err = ghClient.WithEnterpriseURLs(c.endpoint, c.endpoint)
		if err != nil {
			return nil, fmt.Errorf("github: invalid endpoint URL: %w", err)
		}
	}

	c.ghClient = ghClient
	return c, nil
}

func NewClientFromEnv(owner, repo string, opts ...Option) (*httpClient, error) {
	token := os.Getenv("BRAINS_GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("github: BRAINS_GITHUB_TOKEN environment variable not set")
	}
	return NewClient(token, owner, repo, opts...)
}

func (c *httpClient) checkRateLimit(ctx context.Context, resp *gh.Response) error {
	if resp == nil || resp.Rate.Remaining >= c.rateLimitThreshold {
		return nil
	}

	sleepDuration := time.Until(resp.Rate.Reset.Time)
	if sleepDuration <= 0 {
		return nil
	}
	if sleepDuration > maxPreemptiveSleep {
		sleepDuration = maxPreemptiveSleep
	}

	select {
	case <-ctx.Done():
		return NewNetworkError("github: request cancelled during rate limit wait", ctx.Err())
	case <-time.After(sleepDuration):
		return nil
	}
}

func mapError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return NewNetworkError("github: "+err.Error(), err)
	}

	if mapped := mapRateLimitErrors(err); mapped != nil {
		return mapped
	}

	if mapped := mapErrorResponse(err); mapped != nil {
		return mapped
	}

	return NewNetworkError(fmt.Sprintf("github: %s", err.Error()), err)
}

func mapRateLimitErrors(err error) error {
	var rateLimitErr *gh.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return NewRateLimitedError(
			fmt.Sprintf("github: primary rate limit exceeded, resets at %v", rateLimitErr.Rate.Reset),
			err,
		)
	}

	var abuseErr *gh.AbuseRateLimitError
	if errors.As(err, &abuseErr) {
		msg := "github: secondary rate limit exceeded"
		if abuseErr.RetryAfter != nil {
			msg = fmt.Sprintf("%s, retry after %v", msg, *abuseErr.RetryAfter)
		}
		return NewRateLimitedError(msg, err)
	}

	return nil
}

func mapErrorResponse(err error) error {
	var ghErr *gh.ErrorResponse
	if !errors.As(err, &ghErr) {
		return nil
	}

	code := ghErr.Response.StatusCode

	switch {
	case code == http.StatusNotFound:
		return NewNotFoundError(fmt.Sprintf("github: %s", ghErr.Message), err)
	case code == http.StatusTooManyRequests:
		return NewRateLimitedError(fmt.Sprintf("github: rate limited: %s", ghErr.Message), err)
	case code == http.StatusForbidden && ghErr.Response.Header.Get("Retry-After") != "":
		return NewRateLimitedError(fmt.Sprintf("github: rate limited: %s", ghErr.Message), err)
	default:
		return NewAPIError(fmt.Sprintf("github: %s (HTTP %d)", ghErr.Message, code), err)
	}
}

func (c *httpClient) doWithRetry(ctx context.Context, op func() error) error {
	var lastErr error
	for attempt := range maxRetries + 1 {
		err := op()
		if err == nil {
			return nil
		}
		mapped := mapError(err)

		if !IsRateLimited(mapped) && !isServerError(err) {
			return mapped
		}
		lastErr = mapped
		if attempt == maxRetries {
			break
		}

		delay := c.retryDelay(attempt)
		select {
		case <-ctx.Done():
			return NewNetworkError("github: request cancelled during retry", ctx.Err())
		case <-time.After(delay):
		}
	}
	return lastErr
}

func isServerError(err error) bool {
	var ghErr *gh.ErrorResponse
	return errors.As(err, &ghErr) && ghErr.Response.StatusCode >= 500
}

func (c *httpClient) retryDelay(attempt int) time.Duration {
	base := float64(c.retryBase) * math.Pow(2.0, float64(attempt))
	jitter := time.Duration(rand.Int64N(int64(c.retryMaxJitter) + 1))
	return time.Duration(base) + jitter
}

func (c *httpClient) CreatePR(ctx context.Context, input CreatePRInput) (int, error) {
	var prNumber int
	err := c.doWithRetry(ctx, func() error {
		pr, resp, err := c.ghClient.PullRequests.Create(ctx, c.owner, c.repo, &gh.NewPullRequest{
			Title: gh.Ptr(input.Title),
			Body:  gh.Ptr(input.Body),
			Head:  gh.Ptr(input.Head),
			Base:  gh.Ptr(input.Base),
		})
		if err != nil {
			return err
		}
		_ = c.checkRateLimit(ctx, resp)
		prNumber = pr.GetNumber()
		return nil
	})
	if err != nil {
		return 0, err
	}
	return prNumber, nil
}

func (c *httpClient) UpdatePRBody(ctx context.Context, prNumber int, body string) error {
	return c.doWithRetry(ctx, func() error {
		_, resp, err := c.ghClient.PullRequests.Edit(ctx, c.owner, c.repo, prNumber, &gh.PullRequest{
			Body: gh.Ptr(body),
		})
		if err != nil {
			return err
		}
		_ = c.checkRateLimit(ctx, resp)
		return nil
	})
}

func (c *httpClient) GetCommentsSince(ctx context.Context, prNumber int, kind CommentKind, afterID int64) ([]PRComment, error) {
	switch kind {
	case CommentKindIssue:
		return c.getIssueCommentsSince(ctx, prNumber, afterID)
	case CommentKindReview:
		return c.getReviewCommentsSince(ctx, prNumber, afterID)
	default:
		return nil, NewAPIError(fmt.Sprintf("github: unknown comment kind %q", kind), nil)
	}
}

func (c *httpClient) getIssueCommentsSince(ctx context.Context, prNumber int, afterID int64) ([]PRComment, error) {
	var result []PRComment
	opts := &gh.IssueListCommentsOptions{
		Sort:        gh.Ptr("created"),
		Direction:   gh.Ptr("asc"),
		ListOptions: gh.ListOptions{PerPage: 100},
	}

	for {
		var comments []*gh.IssueComment
		var resp *gh.Response
		err := c.doWithRetry(ctx, func() error {
			var e error
			comments, resp, e = c.ghClient.Issues.ListComments(ctx, c.owner, c.repo, prNumber, opts)
			return e
		})
		if err != nil {
			return nil, err
		}

		for _, comment := range comments {
			if comment.GetID() > afterID {
				result = append(result, PRComment{
					ID:        comment.GetID(),
					Author:    comment.GetUser().GetLogin(),
					Body:      comment.GetBody(),
					CreatedAt: comment.GetCreatedAt().Time,
				})
			}
		}

		_ = c.checkRateLimit(ctx, resp)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return result, nil
}

func (c *httpClient) getReviewCommentsSince(ctx context.Context, prNumber int, afterID int64) ([]PRComment, error) {
	var result []PRComment
	opts := &gh.PullRequestListCommentsOptions{
		Sort:        "created",
		Direction:   "asc",
		ListOptions: gh.ListOptions{PerPage: 100},
	}

	for {
		var comments []*gh.PullRequestComment
		var resp *gh.Response
		err := c.doWithRetry(ctx, func() error {
			var e error
			comments, resp, e = c.ghClient.PullRequests.ListComments(ctx, c.owner, c.repo, prNumber, opts)
			return e
		})
		if err != nil {
			return nil, err
		}

		for _, comment := range comments {
			if comment.GetID() > afterID {
				result = append(result, PRComment{
					ID:          comment.GetID(),
					Author:      comment.GetUser().GetLogin(),
					Body:        comment.GetBody(),
					CreatedAt:   comment.GetCreatedAt().Time,
					Path:        comment.GetPath(),
					DiffHunk:    comment.GetDiffHunk(),
					InReplyToID: comment.GetInReplyTo(),
				})
			}
		}

		_ = c.checkRateLimit(ctx, resp)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return result, nil
}

func (c *httpClient) PostCommentReply(ctx context.Context, prNumber int, kind CommentKind, commentID int64, body string) (int64, error) {
	switch kind {
	case CommentKindIssue:
		var newID int64
		err := c.doWithRetry(ctx, func() error {
			comment, resp, err := c.ghClient.Issues.CreateComment(ctx, c.owner, c.repo, prNumber, &gh.IssueComment{
				Body: gh.Ptr(body),
			})
			if err != nil {
				return err
			}
			_ = c.checkRateLimit(ctx, resp)
			newID = comment.GetID()
			return nil
		})
		if err != nil {
			return 0, err
		}
		return newID, nil

	case CommentKindReview:
		if commentID == 0 {
			return 0, NewNotFoundError("github: commentID must be non-zero for review comment replies", nil)
		}
		var newID int64
		err := c.doWithRetry(ctx, func() error {
			comment, resp, err := c.ghClient.PullRequests.CreateCommentInReplyTo(ctx, c.owner, c.repo, prNumber, body, commentID)
			if err != nil {
				return err
			}
			_ = c.checkRateLimit(ctx, resp)
			newID = comment.GetID()
			return nil
		})
		if err != nil {
			return 0, err
		}
		return newID, nil

	default:
		return 0, NewAPIError(fmt.Sprintf("github: unknown comment kind %q", kind), nil)
	}
}

func (c *httpClient) ApplyLabel(ctx context.Context, prNumber int, label string) error {
	return c.doWithRetry(ctx, func() error {
		_, resp, err := c.ghClient.Issues.AddLabelsToIssue(ctx, c.owner, c.repo, prNumber, []string{label})
		if err != nil {
			return err
		}
		_ = c.checkRateLimit(ctx, resp)
		return nil
	})
}

func (c *httpClient) IsMerged(ctx context.Context, prNumber int) (bool, error) {
	var merged bool
	err := c.doWithRetry(ctx, func() error {
		m, resp, err := c.ghClient.PullRequests.IsMerged(ctx, c.owner, c.repo, prNumber)
		if err != nil {
			mapped := mapError(err)
			if IsNotFound(mapped) {
				merged = false
				return nil
			}
			return err
		}
		_ = c.checkRateLimit(ctx, resp)
		merged = m
		return nil
	})
	if err != nil {
		return false, err
	}
	return merged, nil
}

func (c *httpClient) IsClosed(ctx context.Context, prNumber int) (bool, error) {
	var closed bool
	err := c.doWithRetry(ctx, func() error {
		pr, resp, err := c.ghClient.PullRequests.Get(ctx, c.owner, c.repo, prNumber)
		if err != nil {
			return err
		}
		_ = c.checkRateLimit(ctx, resp)
		closed = pr.GetState() == "closed" && !pr.GetMerged()
		return nil
	})
	if err != nil {
		return false, err
	}
	return closed, nil
}

func (c *httpClient) ListOpenPRs(ctx context.Context, label string) ([]PRSummary, error) {
	var result []PRSummary
	opts := &gh.PullRequestListOptions{
		State:       "open",
		ListOptions: gh.ListOptions{PerPage: 100},
	}

	for {
		var prs []*gh.PullRequest
		var resp *gh.Response
		err := c.doWithRetry(ctx, func() error {
			var e error
			prs, resp, e = c.ghClient.PullRequests.List(ctx, c.owner, c.repo, opts)
			return e
		})
		if err != nil {
			return nil, err
		}

		for _, pr := range prs {
			if hasLabel(pr.Labels, label) {
				result = append(result, PRSummary{
					Number: pr.GetNumber(),
					Title:  pr.GetTitle(),
					Head:   pr.GetHead().GetRef(),
					Base:   pr.GetBase().GetRef(),
					Labels: labelNames(pr.Labels),
				})
			}
		}

		_ = c.checkRateLimit(ctx, resp)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return result, nil
}

func hasLabel(labels []*gh.Label, target string) bool {
	for _, l := range labels {
		if l.GetName() == target {
			return true
		}
	}
	return false
}

func labelNames(labels []*gh.Label) []string {
	names := make([]string, len(labels))
	for i, l := range labels {
		names[i] = l.GetName()
	}
	return names
}
