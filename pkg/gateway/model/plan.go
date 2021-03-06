package model

import (
	"time"
)

// Plan is skygear subscription plan which provide information whether the app
// can access the gears
type Plan struct {
	ID          string
	Name        string
	AuthEnabled bool       `db:"auth_enabled"`
	CreatedAt   *time.Time `db:"created_at"`
	UpdatedAt   *time.Time `db:"updated_at"`
}

// CanAccessGear determine whether the app plan can access the given gear
func (p *Plan) CanAccessGear(gear Gear) bool {
	switch gear {
	case AuthGear:
		return p.AuthEnabled
	case AssetGear:
		return true
	default:
		return false
	}
}
