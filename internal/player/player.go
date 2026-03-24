package player

import "time"

type Player struct {
	ID          string    `json:"id" firestore:"id"`
	Email       string    `json:"email" firestore:"email"`
	DisplayName string    `json:"displayName" firestore:"displayName"`
	CreatedAt   time.Time `json:"createdAt" firestore:"createdAt"`
}
