package domain

import (
	"time"

	"github.com/uptrace/bun"
)

// User represents the canonical user entity
type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`
	ID            string    `bun:"id,pk" json:"id"`
	Name          string    `bun:"name,notnull" json:"name"`
	Email         string    `bun:"email,unique,notnull" json:"email"`
	Password      *string   `bun:"password" json:"-"`
	Roles         []string  `bun:"-" json:"roles"`
	IsActive      bool      `bun:"is_active,default:true" json:"is_active"`
	CreatedAt     time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt     time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updated_at"`
}
