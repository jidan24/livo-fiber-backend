package models

import "time"

type Order struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	OrderGineeID     string     `gorm:"uniqueIndex;not null;type:varchar(100)" json:"order_ginee_id"`
	ProcessingStatus string     `gorm:"not null;type:varchar(50);default:ready_to_pick" json:"processing_status"`
	EventStatus      string     `gorm:"not null;type:varchar(50);default:in_progress" json:"event_status"`
	Channel          string     `gorm:"type:varchar(100)" json:"channel"`
	Store            string     `gorm:"type:varchar(100)" json:"store"`
	Buyer            string     `gorm:"type:varchar(150)" json:"buyer"`
	Address          string     `gorm:"type:text" json:"address"`
	Courier          string     `gorm:"type:varchar(100)" json:"courier"`
	TrackingNumber   string     `gorm:"type:varchar(100)" json:"tracking_number"`
	SentBefore       time.Time  `gorm:"type:timestamp;not null" json:"sent_before"`
	AssignedBy       *uint      `gorm:"default:null" json:"assigned_by"`
	AssignedAt       *time.Time `gorm:"default:null" json:"assigned_at"`
	PickedBy         *uint      `gorm:"default:null" json:"picked_by"`
	PickedAt         *time.Time `gorm:"default:null" json:"picked_at"`
	PendingBy        *uint      `gorm:"default:null" json:"pending_by"`
	PendingAt        *time.Time `gorm:"default:null" json:"pending_at"`
	ChangedBy        *uint      `gorm:"default:null" json:"changed_by"`
	ChangedAt        *time.Time `gorm:"default:null" json:"changed_at"`
	DuplicatedBy     *uint      `gorm:"default:null" json:"duplicated_by"`
	DuplicatedAt     *time.Time `gorm:"default:null" json:"duplicated_at"`
	CanceledBy       *uint      `gorm:"default:null" json:"canceled_by"`
	CanceledAt       *time.Time `gorm:"default:null" json:"canceled_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	Complained       bool       `gorm:"default:false" json:"complained"`

	OrderDetails  []OrderDetail `gorm:"foreignKey:OrderID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"order_details,omitempty"`
	AssignUser    *User         `gorm:"foreignKey:AssignedBy" json:"assign_user,omitempty"`
	PickUser      *User         `gorm:"foreignKey:PickedBy" json:"pick_user,omitempty"`
	PendingUser   *User         `gorm:"foreignKey:PendingBy" json:"pending_user,omitempty"`
	ChangeUser    *User         `gorm:"foreignKey:ChangedBy" json:"change_user,omitempty"`
	DuplicateUser *User         `gorm:"foreignKey:DuplicatedBy" json:"duplicate_user,omitempty"`
	CancelUser    *User         `gorm:"foreignKey:CanceledBy" json:"cancel_user,omitempty"`
}

type OrderDetail struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	OrderID     uint   `gorm:"not null" json:"order_id"`
	SKU         string `gorm:"not null;type:varchar(255)" json:"sku"`
	ProductName string `gorm:"not null;type:varchar(255)" json:"product_name"`
	Variant     string `gorm:"type:varchar(100)" json:"variant"`
	Quantity    int    `gorm:"not null" json:"quantity"`
	Price       int    `gorm:"not null" json:"price"`
	IsValid     bool   `gorm:"default:false" json:"is_valid"`
	IsPicked    bool   `gorm:"default:false" json:"is_picked"`

	Order   *Order   `gorm:"foreignKey:OrderID" json:"-"`
	Product *Product `gorm:"-" json:"product,omitempty"`
}

// OrderResponse represents the order data returned in API responses
type OrderResponse struct {
	ID               uint                  `json:"id"`
	OrderGineeID     string                `json:"orderGineeId"`
	ProcessingStatus string                `json:"processingStatus"`
	EventStatus      string                `json:"eventStatus"`
	Channel          string                `json:"channel"`
	Store            string                `json:"store"`
	Buyer            string                `json:"buyer"`
	Address          string                `json:"address"`
	Courier          string                `json:"courier"`
	TrackingNumber   string                `json:"trackingNumber"`
	SentBefore       string                `json:"sentBefore"`
	AssignedBy       *string               `json:"assignedBy,omitempty"`
	AssignedAt       *string               `json:"assignedAt,omitempty"`
	PickedBy         *string               `json:"pickedBy,omitempty"`
	PickedAt         *string               `json:"pickedAt,omitempty"`
	PendingBy        *string               `json:"pendingBy,omitempty"`
	PendingAt        *string               `json:"pendingAt,omitempty"`
	ChangedBy        *string               `json:"changedBy,omitempty"`
	ChangedAt        *string               `json:"changedAt,omitempty"`
	DuplicatedBy     *string               `json:"duplicatedBy,omitempty"`
	DuplicatedAt     *string               `json:"duplicatedAt,omitempty"`
	CanceledBy       *string               `json:"canceledBy,omitempty"`
	CanceledAt       *string               `json:"canceledAt,omitempty"`
	CreatedAt        string                `json:"createdAt"`
	UpdatedAt        string                `json:"updatedAt"`
	Complained       bool                  `json:"complained"`
	Details          []OrderDetailResponse `json:"details,omitempty"`
}

type OrderDetailResponse struct {
	SKU         string `json:"sku"`
	ProductName string `json:"productName"`
	Variant     string `json:"variant"`
	Quantity    int    `json:"quantity"`
	Price       int    `json:"price"`
	IsValid     bool   `json:"isValid"`
	IsPicked    bool   `json:"isPicked"`

	Product *ProductResponse `json:"product,omitempty"`
}

// ToOrderResponse converts an Order model to an OrderResponse
func (o *Order) ToOrderResponse() *OrderResponse {
	details := make([]OrderDetailResponse, len(o.OrderDetails))
	for i, detail := range o.OrderDetails {
		detailResp := OrderDetailResponse{
			SKU:         detail.SKU,
			ProductName: detail.ProductName,
			Variant:     detail.Variant,
			Quantity:    detail.Quantity,
			Price:       detail.Price,
			IsValid:     detail.IsValid,
			IsPicked:    detail.IsPicked,
		}

		// Include product data if exists
		if detail.Product != nil {
			detailResp.Product = &ProductResponse{
				ID:        detail.Product.ID,
				SKU:       detail.Product.SKU,
				Name:      detail.Product.Name,
				Image:     detail.Product.Image,
				Variant:   detail.Product.Variant,
				Location:  detail.Product.Location,
				CreatedAt: detail.Product.CreatedAt.Format("02-01-2006 15:04:05"),
				UpdatedAt: detail.Product.UpdatedAt.Format("02-01-2006 15:04:05"),
			}
		}
		details[i] = detailResp
	}

	// User visual handlers
	var assignedBy, pickedBy, pendingBy, changedBy, duplicatedBy, canceledBy *string
	if o.AssignUser != nil {
		assignedBy = &o.AssignUser.FullName
	}
	if o.PickUser != nil {
		pickedBy = &o.PickUser.FullName
	}
	if o.PendingUser != nil {
		pendingBy = &o.PendingUser.FullName
	}
	if o.ChangeUser != nil {
		changedBy = &o.ChangeUser.FullName
	}
	if o.DuplicateUser != nil {
		duplicatedBy = &o.DuplicateUser.FullName
	}
	if o.CancelUser != nil {
		canceledBy = &o.CancelUser.FullName
	}

	// Date visual handlers
	var assignedAt, pickedAt, pendingAt, changedAt, duplicatedAt, canceledAt *string
	if o.AssignedAt != nil {
		formatted := o.AssignedAt.Format("02-01-2006 15:04:05")
		assignedAt = &formatted
	}
	if o.PickedAt != nil {
		formatted := o.PickedAt.Format("02-01-2006 15:04:05")
		pickedAt = &formatted
	}
	if o.PendingAt != nil {
		formatted := o.PendingAt.Format("02-01-2006 15:04:05")
		pendingAt = &formatted
	}
	if o.ChangedAt != nil {
		formatted := o.ChangedAt.Format("02-01-2006 15:04:05")
		changedAt = &formatted
	}
	if o.DuplicatedAt != nil {
		formatted := o.DuplicatedAt.Format("02-01-2006 15:04:05")
		duplicatedAt = &formatted
	}
	if o.CanceledAt != nil {
		formatted := o.CanceledAt.Format("02-01-2006 15:04:05")
		canceledAt = &formatted
	}

	// Processing status visual handler
	var processingStatus string
	switch o.ProcessingStatus {
	case "ready_to_pick":
		processingStatus = "Ready to Pick"
	case "picking_progress":
		processingStatus = "Picking in Progress"
	case "picking_pending":
		processingStatus = "Picking is Pending"
	case "picking_completed":
		processingStatus = "Picking Completed"
	case "qc_progress":
		processingStatus = "QC in Progress"
	case "qc_pending":
		processingStatus = "QC is Pending"
	case "qc_completed":
		processingStatus = "QC Completed"
	case "outbound_completed":
		processingStatus = "Outbound Completed"
	default:
		processingStatus = o.ProcessingStatus
	}

	// Event status visual handler
	var eventStatus string
	switch o.EventStatus {
	case "in_progress":
		eventStatus = "In Progress"
	case "completed":
		eventStatus = "Completed"
	case "pending":
		eventStatus = "Pending"
	case "canceled", "cancelled":
		eventStatus = "Canceled"
	case "duplicated":
		eventStatus = "Duplicated"
	default:
		eventStatus = o.EventStatus
	}

	return &OrderResponse{
		ID:               o.ID,
		OrderGineeID:     o.OrderGineeID,
		ProcessingStatus: processingStatus,
		EventStatus:      eventStatus,
		Channel:          o.Channel,
		Store:            o.Store,
		Buyer:            o.Buyer,
		Address:          o.Address,
		Courier:          o.Courier,
		TrackingNumber:   o.TrackingNumber,
		SentBefore:       o.SentBefore.Format("02-01-2006 15:04:05"),
		AssignedBy:       assignedBy,
		AssignedAt:       assignedAt,
		PickedBy:         pickedBy,
		PickedAt:         pickedAt,
		PendingBy:        pendingBy,
		PendingAt:        pendingAt,
		ChangedBy:        changedBy,
		ChangedAt:        changedAt,
		DuplicatedBy:     duplicatedBy,
		DuplicatedAt:     duplicatedAt,
		CanceledBy:       canceledBy,
		CanceledAt:       canceledAt,
		CreatedAt:        o.CreatedAt.Format("02-01-2006 15:04:05"),
		UpdatedAt:        o.UpdatedAt.Format("02-01-2006 15:04:05"),
		Complained:       o.Complained,
		Details:          details,
	}
}
