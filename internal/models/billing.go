package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"cirrussync-api/internal/utils"
)

// VendorStates represents the status of different payment vendors
type AvailablePaymentMethods struct {
	Apple int `json:"Apple"`
	Card  int `json:"Card"`
	InApp int `json:"InApp"`
}

// BillingInfo represents the billing and payment information for a user
type BillingInfo struct {
	ID                      string                  `gorm:"primaryKey;column:id" json:"id"`
	UserID                  string                  `gorm:"column:user_id;not null;index:idx_billing_info_user_id" json:"user_id"`
	CountryCode             string                  `gorm:"column:country_code;size:2" json:"country_code"`
	State                   string                  `gorm:"column:state;size:50" json:"state"`
	ZipCode                 *string                 `gorm:"column:zip_code" json:"zip_code"`
	AvailablePaymentMethods AvailablePaymentMethods `gorm:"column:available_payment_methods;type:jsonb;serializer:json" json:"available_payment_methods"`
	CreatedAt               int64                   `gorm:"column:created_at;autoCreateTime:false;not null" json:"created_at"`
	UpdatedAt               int64                   `gorm:"column:updated_at;autoUpdateTime:false;not null" json:"updated_at"`
	DeletedAt               gorm.DeletedAt          `gorm:"column:deleted_at,omitempty"`

	// // Relationships
	// User           User                `gorm:"foreignKey:UserID"`
	// PaymentMethods []UserPaymentMethod `gorm:"foreignKey:UserID;references:UserID" json:"payment_methods,omitempty"`
	// BillingRecords []UserBilling       `gorm:"foreignKey:UserID;references:UserID" json:"billing_records,omitempty"`
	// UserPlans      []UserPlan          `gorm:"foreignKey:UserID;references:UserID" json:"user_plans,omitempty"`
}

// TableName specifies the table name for BillingInfo
func (BillingInfo) TableName() string {
	return "billing_info"
}

// BeforeCreate will set timestamps and generate ID before creating a record
func (b *BillingInfo) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if b.ID == "" {
		b.ID = utils.GenerateLinkID()
	}
	if b.CreatedAt == 0 {
		b.CreatedAt = now
	}
	if b.UpdatedAt == 0 {
		b.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate will update timestamps before updating a record
func (b *BillingInfo) BeforeUpdate(tx *gorm.DB) error {
	b.UpdatedAt = time.Now().Unix()
	return nil
}

// Plan represents a subscription plan
type Plan struct {
	ID           string  `gorm:"primaryKey;column:id"`
	PlanType     string  `gorm:"column:plan_type;size:20;not null;index:idx_plans_plan_type"`
	Name         string  `gorm:"column:name;size:100;not null"`
	Description  string  `gorm:"column:description;size:200"`
	Tier         int     `gorm:"column:tier;not null;index:idx_plans_tier"`
	Features     string  `gorm:"column:features;type:text;not null"`
	MonthlyPrice float64 `gorm:"column:monthly_price;not null"`
	YearlyPrice  float64 `gorm:"column:yearly_price;not null"`
	StorageQuota int64   `gorm:"column:storage_quota;not null"`
	MaxDevices   int     `gorm:"column:max_devices;default:0"`
	MaxUsers     int     `gorm:"column:max_users;default:1"`
	Available    bool    `gorm:"column:available;default:true;index:idx_plans_available"`
	Popular      bool    `gorm:"column:popular;default:false"`
	CreatedAt    int64   `gorm:"column:created_at;autoCreateTime:false;not null"`
	UpdatedAt    int64   `gorm:"column:updated_at;autoUpdateTime:false;not null"`

	// Relationships
	UserPlans []UserPlan `gorm:"foreignKey:PlanID"`
}

// TableName specifies the table name for Plan
func (Plan) TableName() string {
	return "plans"
}

// BeforeCreate will set timestamps before creating a record
func (p *Plan) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if p.CreatedAt == 0 {
		p.CreatedAt = now
	}
	if p.UpdatedAt == 0 {
		p.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate will update timestamps before updating a record
func (p *Plan) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now().Unix()
	return nil
}

// UserPlan represents the association between users and plans
type UserPlan struct {
	ID                  string `gorm:"primaryKey;column:id"`
	UserID              string `gorm:"column:user_id;not null;index:idx_users_plans_user_id"`
	PlanID              string `gorm:"column:plan_id;size:10;not null;index:idx_users_plans_plan_id"`
	PlanType            string `gorm:"column:plan_type;size:20;default:'individual'"`
	Status              string `gorm:"column:status;size:20;default:'active';index:idx_users_plans_status"`
	BillingCycle        string `gorm:"column:billing_cycle;size:10;default:'monthly'"`
	AutoRenew           bool   `gorm:"column:auto_renew;default:true"`
	TrialStart          int64  `gorm:"column:trial_start"`
	TrialEnd            int64  `gorm:"column:trial_end"`
	CurrentPeriodStart  int64  `gorm:"column:current_period_start;not null;index:idx_users_plans_period,priority:1"`
	CurrentPeriodEnd    int64  `gorm:"column:current_period_end;index:idx_users_plans_period,priority:2"`
	StorageQuota        int64  `gorm:"column:storage_quota;not null"`
	AdditionalStorage   int64  `gorm:"column:additional_storage;default:0"`
	Delinquent          bool   `gorm:"column:delinquent;default:false"`
	DelinquentAt        int64  `gorm:"column:delinquent_at"`
	DelinquentReason    string `gorm:"column:delinquent_reason;size:100"`
	ScheduledDeletionAt int64  `gorm:"column:scheduled_deletion_at"`
	CanceledAt          int64  `gorm:"column:canceled_at"`
	CancellationReason  string `gorm:"column:cancellation_reason;size:100"`
	ExternalReference   string `gorm:"column:external_reference;size:100;index:idx_users_plans_ext_ref"`
	CreatedAt           int64  `gorm:"column:created_at;autoCreateTime:false;not null"`
	ModifiedAt          int64  `gorm:"column:modified_at;autoCreateTime:false;not null"`

	// Relationships
	User           User          `gorm:"foreignKey:UserID"`
	Plan           Plan          `gorm:"foreignKey:PlanID"`
	BillingRecords []UserBilling `gorm:"foreignKey:PlanID"`
}

// TableName specifies the table name for UserPlan
func (UserPlan) TableName() string {
	return "users_plans"
}

// BeforeCreate hook for UserPlan
func (up *UserPlan) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if up.ID == "" {
		up.ID = utils.GenerateLinkID()
	}
	if up.CreatedAt == 0 {
		up.CreatedAt = now
	}
	if up.ModifiedAt == 0 {
		up.ModifiedAt = now
	}
	return nil
}

// BeforeUpdate hook for UserPlan
func (up *UserPlan) BeforeUpdate(tx *gorm.DB) error {
	up.ModifiedAt = time.Now().Unix()
	return nil
}

// UserBilling represents billing records for users
type UserBilling struct {
	ID                 string          `gorm:"primaryKey;column:id"`
	UserID             string          `gorm:"column:user_id;not null;index:idx_billing_user_id"`
	PlanID             string          `gorm:"column:plan_id;index:idx_billing_plan_id"`
	Amount             float64         `gorm:"column:amount;not null"`
	Currency           string          `gorm:"column:currency;size:3;default:'USD'"`
	Status             string          `gorm:"column:status;size:20;not null;index:idx_billing_status"`
	Description        string          `gorm:"column:description;size:200"`
	PeriodStart        int64           `gorm:"column:period_start;index:idx_billing_period,priority:1"`
	PeriodEnd          int64           `gorm:"column:period_end;index:idx_billing_period,priority:2"`
	PaymentMethodType  string          `gorm:"column:payment_method_type;size:50"`
	PaymentMethodLast4 string          `gorm:"column:payment_method_last4;size:4"`
	PaymentMethodBrand string          `gorm:"column:payment_method_brand;size:20"`
	InvoiceURL         string          `gorm:"column:invoice_url;size:255"`
	ReceiptURL         string          `gorm:"column:receipt_url;size:255"`
	AdditionalMetadata json.RawMessage `gorm:"column:additional_metadata;type:jsonb"`
	ExternalReference  string          `gorm:"column:external_reference;size:100;index:idx_billing_ext_ref"`
	CreatedAt          int64           `gorm:"column:created_at;autoCreateTime:false;not null;index:idx_billing_created_at"`
	UpdatedAt          int64           `gorm:"column:updated_at;autoCreateTime:false;not null"`

	// Relationships
	User User     `gorm:"foreignKey:UserID"`
	Plan UserPlan `gorm:"foreignKey:PlanID"`
}

// TableName specifies the table name for UserBilling
func (UserBilling) TableName() string {
	return "users_billing"
}

// BeforeCreate hook for UserBilling
func (ub *UserBilling) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if ub.ID == "" {
		ub.ID = utils.GenerateLinkID()
	}
	if ub.CreatedAt == 0 {
		ub.CreatedAt = now
	}
	if ub.UpdatedAt == 0 {
		ub.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate hook for UserBilling
func (ub *UserBilling) BeforeUpdate(tx *gorm.DB) error {
	ub.UpdatedAt = time.Now().Unix()
	return nil
}

// UserPaymentMethod represents payment methods for users
type UserPaymentMethod struct {
	ID                 string          `gorm:"primaryKey;column:id"`
	UserID             string          `gorm:"column:user_id;not null;index:idx_payment_methods_user_id"`
	Type               string          `gorm:"column:type;size:50;not null"`
	IsDefault          bool            `gorm:"column:is_default;default:false"`
	Last4              string          `gorm:"column:last4;size:4"`
	Brand              string          `gorm:"column:brand;size:20"`
	ExpMonth           int             `gorm:"column:exp_month"`
	ExpYear            int             `gorm:"column:exp_year"`
	BillingName        string          `gorm:"column:billing_name;size:100"`
	BillingAddress     string          `gorm:"column:billing_address;type:text"`
	BillingCountry     string          `gorm:"column:billing_country;size:2"`
	BillingPostalCode  string          `gorm:"column:billing_postal_code;size:20"`
	AdditionalMetadata json.RawMessage `gorm:"column:additional_metadata;type:jsonb"`
	ExternalReference  string          `gorm:"column:external_reference;size:100;index:idx_payment_methods_ext_ref"`
	CreatedAt          int64           `gorm:"column:created_at;autoCreateTime:false;not null"`
	UpdatedAt          int64           `gorm:"column:updated_at;autoCreateTime:false;not null"`
	LastUsed           int64           `gorm:"column:last_used"`
	Active             bool            `gorm:"column:active;default:true;index:idx_payment_methods_active"`

	// Relationships
	User User `gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for UserPaymentMethod
func (UserPaymentMethod) TableName() string {
	return "users_payment_methods"
}

// BeforeCreate hook for UserPaymentMethod
func (upm *UserPaymentMethod) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if upm.ID == "" {
		upm.ID = utils.GenerateLinkID()
	}
	if upm.CreatedAt == 0 {
		upm.CreatedAt = now
	}
	if upm.UpdatedAt == 0 {
		upm.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate hook for UserPaymentMethod
func (upm *UserPaymentMethod) BeforeUpdate(tx *gorm.DB) error {
	upm.UpdatedAt = time.Now().Unix()
	return nil
}

// GiftCard represents gift cards
type GiftCard struct {
	ID                 string          `gorm:"primaryKey;column:id;size:36"`
	Value              string          `gorm:"column:value;size:64;not null;unique;index"`
	Amount             int             `gorm:"column:amount;not null"`
	Currency           string          `gorm:"column:currency;size:3;not null;default:'USD'"`
	CreatedBy          string          `gorm:"column:created_by;size:36"`
	CreatedAt          int64           `gorm:"column:created_at;not null"`
	ExpirationDate     int64           `gorm:"column:expiration_date;index:idx_gift_cards_expiration"`
	IsUsed             bool            `gorm:"column:is_used;not null;default:false;index"`
	UsedBy             string          `gorm:"column:used_by;size:36;index:idx_gift_cards_user_status,priority:1"`
	UsedAt             int64           `gorm:"column:used_at"`
	AdditionalMetadata json.RawMessage `gorm:"column:additional_metadata;type:jsonb"`

	// Relationships
	Creator  User `gorm:"foreignKey:CreatedBy"`
	Redeemer User `gorm:"foreignKey:UsedBy"`
}

// TableName specifies the table name for GiftCard
func (GiftCard) TableName() string {
	return "gift_cards"
}
