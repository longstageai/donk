package creative

import (
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// nextID 生成带业务前缀的唯一 ID，便于调试事件流。
func nextID(prefix string) ID {
	return NextID(prefix)
}

// NextID 生成带业务前缀的唯一 ID，供 creative 子包创建 Artifact 等对象时复用。
func NextID(prefix string) ID {
	return ID(prefix + "_" + uuid.NewString())
}

// Store 是第一阶段内存存储实现，集中保存事件、消息、产物、AgentRun 和快照。
type Store struct {
	mu        sync.RWMutex           // 读写锁，保证并发安全
	sessions  map[ID]Session         // Session存储映射
	rooms     map[ID]Room            // Room存储映射
	events    map[ID]Event           // Event存储映射
	messages  map[ID]Message         // Message存储映射
	artifacts map[ID]Artifact        // Artifact存储映射
	runs      map[ID]AgentRun        // AgentRun存储映射
	snapshots map[ID][]StateSnapshot // 状态快照存储映射
}

// NewStore 创建内存存储。
func NewStore() *Store {
	return &Store{
		sessions:  map[ID]Session{},
		rooms:     map[ID]Room{},
		events:    map[ID]Event{},
		messages:  map[ID]Message{},
		artifacts: map[ID]Artifact{},
		runs:      map[ID]AgentRun{},
		snapshots: map[ID][]StateSnapshot{},
	}
}

func (s *Store) SaveSession(session Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
}

func (s *Store) GetSession(id ID) (Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[id]
	return session, ok
}

func (s *Store) LatestSession() (Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var latest Session
	for _, session := range s.sessions {
		if latest.ID == "" || session.StartedAt.After(latest.StartedAt) {
			latest = session
		}
	}
	return latest, latest.ID != ""
}

func (s *Store) UpdateSession(id ID, fn func(*Session)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[id]
	if !ok {
		return false
	}
	fn(&session)
	s.sessions[id] = session
	return true
}

func (s *Store) SaveRoom(room Room) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rooms[room.ID] = room
}

func (s *Store) GetRoom(id ID) (Room, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	room, ok := s.rooms[id]
	return room, ok
}

func (s *Store) SaveEvent(event Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}
	event.UpdatedAt = now
	s.events[event.ID] = event
}

func (s *Store) GetEvent(id ID) (Event, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	event, ok := s.events[id]
	return event, ok
}

func (s *Store) UpdateEvent(id ID, fn func(*Event)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	event, ok := s.events[id]
	if !ok {
		return false
	}
	fn(&event)
	event.UpdatedAt = time.Now()
	s.events[id] = event
	return true
}

// ListPendingEvents 返回指定 Session 下所有待处理事件，并按优先级和创建时间排序。
func (s *Store) ListPendingEvents(sessionID ID) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Event, 0)
	for _, event := range s.events {
		if event.SessionID == sessionID && event.Status == EventPending {
			result = append(result, event)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Priority == result[j].Priority {
			return result[i].CreatedAt.Before(result[j].CreatedAt)
		}
		return result[i].Priority > result[j].Priority
	})
	return result
}

// ListEvents 返回指定 Session 的全部事件。
func (s *Store) ListEvents(sessionID ID) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.eventsForSessionLocked(sessionID)
}

// eventsForSessionLocked 必须在持有读锁或写锁时调用。
func (s *Store) eventsForSessionLocked(sessionID ID) []Event {
	result := make([]Event, 0)
	for _, event := range s.events {
		if event.SessionID == sessionID {
			result = append(result, event)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.Before(result[j].CreatedAt) })
	return result
}

func (s *Store) SaveMessage(message Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages[message.ID] = message
}

func (s *Store) ListMessages(roomID ID, limit int) []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Message, 0)
	for _, message := range s.messages {
		if message.RoomID == roomID {
			items = append(items, message)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	if limit > 0 && len(items) > limit {
		items = items[len(items)-limit:]
	}
	return items
}

func (s *Store) SaveArtifact(artifact Artifact) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.artifacts[artifact.ID] = artifact
}

func (s *Store) ListArtifacts(sessionID ID) []Artifact {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Artifact, 0)
	for _, artifact := range s.artifacts {
		if artifact.SessionID == sessionID {
			items = append(items, artifact)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items
}

func (s *Store) SaveAgentRun(run AgentRun) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[run.ID] = run
}

func (s *Store) UpdateAgentRun(id ID, fn func(*AgentRun)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.runs[id]
	if !ok {
		return false
	}
	fn(&run)
	s.runs[id] = run
	return true
}

func (s *Store) ListAgentRuns(sessionID ID) []AgentRun {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]AgentRun, 0)
	for _, run := range s.runs {
		if run.SessionID == sessionID {
			items = append(items, run)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].StartedAt.Before(items[j].StartedAt) })
	return items
}

func (s *Store) SaveSnapshot(snapshot StateSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshots[snapshot.SessionID] = append(s.snapshots[snapshot.SessionID], snapshot)
}

func (s *Store) ListSnapshots(sessionID ID) []StateSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := append([]StateSnapshot(nil), s.snapshots[sessionID]...)
	return items
}
