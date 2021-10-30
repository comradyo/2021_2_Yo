package session

import (
	"backend/session/models"
	"backend/session/repository"
	"math/rand"
	"time"
)

type Manager struct {
	repository repository.Repository
}

func NewManager(repo repository.Repository) *Manager {
	return &Manager{
		repository: repo,
	}
}

const (
	sessionIdLength = 16
	sessionLifeTime = time.Hour * 24
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func generateSessionId(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func (m *Manager) Create(userId string) (string, error) {
	sessionData := &models.SessionData{
		SessionId:  generateSessionId(sessionIdLength),
		UserId:     userId,
		Expiration: int(sessionLifeTime.Milliseconds()),
	}
	err := m.repository.Create(sessionData)
	if err != nil {
		return "", err
	}
	return sessionData.SessionId, nil
}

func (m *Manager) Check(sessionId string) (string, error) {
	userId, err := m.repository.Check(sessionId)
	if err != nil {
		return "", err
	}
	return userId, nil
}

func (m *Manager) Delete(sessionId string) error {
	err := m.repository.Delete(sessionId)
	if err != nil {
		return err
	}
	return nil
}
