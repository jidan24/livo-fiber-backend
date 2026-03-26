package models

import "time"

type Return struct {
	ID                uint   `gorm:"primaryKey" json:"id"`
	NewTrackingNumber string `gorm:"not null;index;type:varchar(255)" json:"new_tracking_number"`

	ChannelID      uint      `gorm:"not null" json:"channel_id"`
	StoreID        uint      `gorm:"not null" json:"store_id"`
	CreatedBy      uint      `gorm:"not null" json:"created_by"`
	UpdatedBy      *uint     `gorm:"default:null" json:"updated_by"`
	OrderGineeID   *string   `gorm:"default:null;type:varchar(255)" json:"order_ginee_id"`
	TrackingNumber *string   `gorm:"default:null;index;type:varchar(255)" json:"tracking_number"`
	ReturnType     *string   `gorm:"default:null;type:varchar(100)" json:"return_type"`
	ReturnReason   *string   `gorm:"default:null;type:text" json:"return_reason"`
	ReturnNumber   *string   `gorm:"default:null;type:varchar(20)" json:"return_number"`
	ScrapNumber    *string   `gorm:"default:null;type:varchar(20)" json:"scrap_number"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	ReturnDetails *[]ReturnDetail `gorm:"foreignKey:ReturnID" json:"return_details,omitempty"`
	CreateUser    *User           `gorm:"foreignKey:CreatedBy" json:"create_user,omitempty"`
	UpdateUser    *User           `gorm:"foreignKey:UpdatedBy" json:"update_user,omitempty"`
	Store         *Store          `gorm:"foreignKey:StoreID" json:"store,omitempty"`
	Channel       *Channel        `gorm:"foreignKey:ChannelID" json:"channel,omitempty"`
	Order         *Order          `gorm:"-" json:"order,omitempty"`
}

type ReturnDetail struct {
	ID         *uint   `gorm:"primaryKey" json:"id"`
	ReturnID   *uint   `gorm:"not null" json:"return_id"`
	ProductSKU *string `gorm:"not null;type:varchar(255)" json:"product_sku"`
	Quantity   *int    `gorm:"not null" json:"quantity"`
	Price      *int    `gorm:"not null" json:"price"`

	Return  Return   `gorm:"foreignKey:ReturnID" json:"-"`
	Product *Product `gorm:"-" json:"product,omitempty"`
}

// Response structs and ToResponse methods can be added similarly as in LostFound model
type ReturnResponse struct {
	ID                uint                    `json:"id"`
	NewTrackingNumber string                  `json:"newTrackingNumber"`
	OrderGineeID      *string                 `json:"orderGineeId"`
	Channel           string                  `json:"channel"`
	Store             string                  `json:"store"`
	CreatedBy         string                  `json:"createdBy"`
	UpdatedBy         *string                 `json:"updatedBy,omitempty"`
	TrackingNumber    *string                 `json:"trackingNumber,omitempty"`
	ReturnType        *string                 `json:"returnType,omitempty"`
	ReturnReason      *string                 `json:"returnReason,omitempty"`
	ReturnNumber      *string                 `json:"returnNumber,omitempty"`
	ScrapNumber       *string                 `json:"scrapNumber,omitempty"`
	CreatedAt         string                  `json:"createdAt"`
	UpdatedAt         string                  `json:"updatedAt"`
	Details           *[]ReturnDetailResponse `json:"details,omitempty"`
	Order             *OrderResponse          `json:"order,omitempty"`
}

type ReturnDetailResponse struct {
	ProductSKU *string          `json:"productSKU"`
	Quantity   *int             `json:"quantity"`
	Price      *int             `json:"price"`
	Product    *ProductResponse `json:"product,omitempty"`
}

type MobileReturnResponse struct {
	ID                uint   `json:"id"`
	NewTrackingNumber string `json:"newTrackingNumber"`
	ChannelID         string `json:"channelId"`
	StoreID           string `json:"storeId"`
	CreatedBy         string `json:"createdBy"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
}

// ToResponse converts Return model to ReturnResponse
func (r *Return) ToResponse() ReturnResponse {
	// Convert Return Details
	var details []ReturnDetailResponse
	if r.ReturnDetails != nil {
		details = make([]ReturnDetailResponse, len(*r.ReturnDetails))
		for i, detail := range *r.ReturnDetails {
			detailResp := ReturnDetailResponse{
				ProductSKU: detail.ProductSKU,
				Quantity:   detail.Quantity,
				Price:      detail.Price,
			}
			// Only add product response if product exists
			if detail.Product != nil {
				productResp := detail.Product.ToResponse()
				detailResp.Product = productResp
			}
			details[i] = detailResp
		}
	}

	// Channel and Store names
	var channelName string
	if r.Channel != nil {
		channelName = r.Channel.ChannelName
	}

	var storeName string
	if r.Store != nil {
		storeName = r.Store.StoreName
	}

	// User visual handlers
	var createdBy string
	if r.CreateUser != nil {
		createdBy = r.CreateUser.FullName
	}

	var updatedBy *string
	if r.UpdateUser != nil {
		updatedBy = &r.UpdateUser.FullName
	}

	// Include Order response if tracking number exists in Order
	var orderResponse *OrderResponse
	if r.Order != nil {
		orderResp := r.Order.ToOrderResponse()
		orderResponse = orderResp
	}

	return ReturnResponse{
		ID:                r.ID,
		NewTrackingNumber: r.NewTrackingNumber,
		OrderGineeID:      r.OrderGineeID,
		Channel:           channelName,
		Store:             storeName,
		CreatedBy:         createdBy,
		UpdatedBy:         updatedBy,
		TrackingNumber:    r.TrackingNumber,
		ReturnType:        r.ReturnType,
		ReturnReason:      r.ReturnReason,
		ReturnNumber:      r.ReturnNumber,
		ScrapNumber:       r.ScrapNumber,
		CreatedAt:         r.CreatedAt.Format("02-01-2006 15:04:05"),
		UpdatedAt:         r.UpdatedAt.Format("02-01-2006 15:04:05"),
		Details:           &details,
		Order:             orderResponse,
	}
}

func (r *Return) ToMobileResponse() MobileReturnResponse {
	// Channel and Store names
	var channelName string
	if r.Channel != nil {
		channelName = r.Channel.ChannelName
	}

	var storeName string
	if r.Store != nil {
		storeName = r.Store.StoreName
	}

	// User visual handlers
	var createdBy string
	if r.CreateUser != nil {
		createdBy = r.CreateUser.FullName
	}

	return MobileReturnResponse{
		ID:                r.ID,
		NewTrackingNumber: r.NewTrackingNumber,
		ChannelID:         channelName,
		StoreID:           storeName,
		CreatedBy:         createdBy,
		CreatedAt:         r.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:         r.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
