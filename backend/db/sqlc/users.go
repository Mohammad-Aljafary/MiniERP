package models

import "time"

type User struct {
	ID           string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email        string         `gorm:"not null;uniqueIndex" json:"email"`
	PasswordHash string         `gorm:"not null" json:"-"`
	DisplayName  string         `gorm:"not null" json:"display_name"`
	Status       string         `gorm:"not null;check:status IN ('active', 'disabled', 'pending')" json:"status"`
	CreatedAt    time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"not null" json:"updated_at"`
}

type Tenant struct {
	ID              string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name            string    `gorm:"not null" json:"name"`
	Slug            string    `gorm:"not null;uniqueIndex" json:"slug"`
	Status          string    `gorm:"not null;check:status IN ('trial', 'active', 'suspended', 'closed')" json:"status"`
	DefaultCurrency string    `gorm:"type:char(3);not null" json:"default_currency"`
	Timezone        string    `gorm:"not null" json:"timezone"`
	CreatedAt       time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt       time.Time `gorm:"not null" json:"updated_at"`
}

type TenantMembership struct {
	ID        string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID  string     `gorm:"type:uuid;not null;uniqueIndex:uidx_tenant_memberships_tenant_user,priority:1" json:"tenant_id"`
	UserID    string     `gorm:"type:uuid;not null;uniqueIndex:uidx_tenant_memberships_tenant_user,priority:2;index:idx_tenant_memberships_user_status,priority:1" json:"user_id"`
	RoleID    string     `gorm:"type:uuid;not null" json:"role_id"`
	Status    string     `gorm:"not null;check:status IN ('invited', 'active', 'suspended');index:idx_tenant_memberships_user_status,priority:2" json:"status"`
	JoinedAt  *time.Time `json:"joined_at"`
	CreatedAt time.Time  `gorm:"not null" json:"created_at"`

	Tenant Tenant `gorm:"foreignKey:TenantID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"-"`
	User   User   `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"-"`
}

func (TenantMembership) TableName() string {
	return "tenant_memberships"
}
