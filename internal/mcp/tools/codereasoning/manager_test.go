package codereasoning

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestManager_GetOrCreate_NewSession(t *testing.T) {
	m := NewSessionManager()

	s := m.GetOrCreate("session-1")
	assert.NotNil(t, s)
	assert.Equal(t, 1, m.Count())
}

func TestManager_GetOrCreate_ExistingSession(t *testing.T) {
	m := NewSessionManager()

	s1 := m.GetOrCreate("session-1")
	s1.AddThought(ThoughtRequest{
		Thought:           "First thought",
		ThoughtNumber:     1,
		TotalThoughts:     3,
		NextThoughtNeeded: true,
	})

	s2 := m.GetOrCreate("session-1")

	// Should be the same session
	assert.Equal(t, s1.GetCurrentThoughtNumber(), s2.GetCurrentThoughtNumber())
	assert.Equal(t, 1, m.Count())
}

func TestManager_Get_NonExistent(t *testing.T) {
	m := NewSessionManager()

	s := m.Get("nonexistent")
	assert.Nil(t, s)
}

func TestManager_Delete(t *testing.T) {
	m := NewSessionManager()

	m.GetOrCreate("session-1")
	assert.Equal(t, 1, m.Count())

	m.Delete("session-1")
	assert.Equal(t, 0, m.Count())
	assert.Nil(t, m.Get("session-1"))
}

func TestManager_Cleanup_RemovesOldSessions(t *testing.T) {
	m := NewSessionManager()

	// Create some sessions
	m.GetOrCreate("session-1")
	m.GetOrCreate("session-2")
	m.GetOrCreate("session-3")

	assert.Equal(t, 3, m.Count())

	// Cleanup with 0 max age should remove all
	removed := m.Cleanup(0)
	assert.Equal(t, 3, removed)
	assert.Equal(t, 0, m.Count())
}

func TestManager_Cleanup_KeepsRecentSessions(t *testing.T) {
	m := NewSessionManager()

	m.GetOrCreate("session-1")

	// Cleanup with 1 hour max age should keep recent session
	removed := m.Cleanup(time.Hour)
	assert.Equal(t, 0, removed)
	assert.Equal(t, 1, m.Count())
}

func TestManager_MultipleSessions(t *testing.T) {
	m := NewSessionManager()

	s1 := m.GetOrCreate("session-1")
	s2 := m.GetOrCreate("session-2")

	// Modify different sessions
	s1.AddThought(ThoughtRequest{
		Thought:           "Session 1 thought",
		ThoughtNumber:     1,
		TotalThoughts:     2,
		NextThoughtNeeded: true,
	})

	s2.AddThought(ThoughtRequest{
		Thought:           "Session 2 thought",
		ThoughtNumber:     1,
		TotalThoughts:     3,
		NextThoughtNeeded: true,
	})

	// Verify they are independent
	assert.Equal(t, 2, m.GetOrCreate("session-1").GetTotalThoughts())
	assert.Equal(t, 3, m.GetOrCreate("session-2").GetTotalThoughts())
}

func TestManager_ThreadSafe(t *testing.T) {
	m := NewSessionManager()
	done := make(chan bool)

	// Concurrent operations
	for i := 0; i < 10; i++ {
		go func(id int) {
			sessionID := "session-" + string(rune('A'+id))
			m.GetOrCreate(sessionID)
			m.Get(sessionID)
			m.Count()
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic or deadlock
	assert.True(t, m.Count() > 0)
}
