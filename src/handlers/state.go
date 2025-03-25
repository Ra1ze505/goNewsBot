package handlers

type UserState struct {
	ChangingCity bool
}

type StateStorage struct {
	state map[int64]*UserState
}

func NewStateStorage() *StateStorage {
	return &StateStorage{
		state: make(map[int64]*UserState),
	}
}

func (s *StateStorage) GetState(userID int64) *UserState {
	return s.state[userID]
}

func (s *StateStorage) SetState(userID int64, state *UserState) {
	s.state[userID] = state
}

func (s *StateStorage) ClearState(userID int64) {
	delete(s.state, userID)
}
