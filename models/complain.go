package models

import "time"

type Complain struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Code           string    `gorm:"not null;uniqueIndex;type:varchar(50)" json:"code"`
	TrackingNumber string    `gorm:"not null;uniqueIndex;type:varchar(100)" json:"tracking_number"`
	OrderGineeID   string    `gorm:"not null;index;type:varchar(100)" json:"order_ginee_id"`
	ChannelID      uint      `gorm:"not null" json:"channel_id"`
	StoreID        uint      `gorm:"not null" json:"store_id"`
	CreatedBy      uint      `gorm:"not null" json:"created_by"`
	Reason         string    `gorm:"not null;type:text" json:"reason"`
	Solution       *string   `gorm:"default:null;type:text" json:"solution"`
	TotalFee       *int      `gorm:"default:null" json:"total_fee"`
	Checked        bool      `gorm:"default:false" json:"checked"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	ComplainProductDetails []ComplainProductDetail `gorm:"foreignKey:ComplainID" json:"complain_product_details,omitempty"`
	ComplainUserDetails    []ComplainUserDetail    `gorm:"foreignKey:ComplainID" json:"complain_user_details,omitempty"`
	Channel                *Channel                `gorm:"foreignKey:ChannelID" json:"channel,omitempty"`
	Store                  *Store                  `gorm:"foreignKey:StoreID" json:"store,omitempty"`
	CreateUser             *User                   `gorm:"foreignKey:CreatedBy" json:"create_user,omitempty"`
	Order                  *Order                  `gorm:"-" json:"order,omitempty"`
	Return                 *Return                 `gorm:"-" json:"return,omitempty"`
}

type ComplainProductDetail struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	ComplainID uint   `gorm:"not null" json:"complain_id"`
	ProductSKU string `gorm:"not null;type:varchar(255)" json:"product_sku"`
	Quantity   int    `gorm:"not null" json:"quantity"`
	Price      int    `gorm:"not null" json:"price"`

	Complain Complain `gorm:"foreignKey:ComplainID" json:"-"`
	Product  *Product `gorm:"-" json:"product,omitempty"`
}

type ComplainUserDetail struct {
	ID         uint `gorm:"primaryKey" json:"id"`
	ComplainID uint `gorm:"not null" json:"complain_id"`
	UserID     uint `gorm:"not null" json:"user_id"`
	FeeCharge  int  `gorm:"not null" json:"fee_charge"`

	Complain Complain `gorm:"foreignKey:ComplainID" json:"-"`
	User     *User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// ComplainResponse represents the complain data returned in API responses
type ComplainResponse struct {
	ID             uint                            `json:"id"`
	Code           string                          `json:"code"`
	TrackingNumber string                          `json:"trackingNumber"`
	Channel        string                          `json:"channel"`
	Store          string                          `json:"store"`
	Reason         string                          `json:"reason"`
	CreatedBy      string                          `json:"createdBy"`
	Solution       *string                         `json:"solution,omitempty"`
	TotalFee       *int                            `json:"totalFee,omitempty"`
	OrderGineeID   string                          `json:"orderGineeId"`
	Checked        bool                            `json:"checked"`
	CreatedAt      string                          `json:"createdAt"`
	UpdatedAt      string                          `json:"updatedAt"`
	ProductDetails []ComplainProductDetailResponse `json:"details,omitempty"`
	UserDetails    []ComplainUserDetailResponse    `json:"userDetails,omitempty"`
}

type ComplainProductDetailResponse struct {
	ProductSKU string           `json:"productSKU"`
	Quantity   int              `json:"quantity"`
	Price      int              `json:"price"`
	Product    *ProductResponse `json:"product,omitempty"`
}

type ComplainUserDetailResponse struct {
	ID        uint   `json:"id"`
	User      string `json:"user"`
	FeeCharge int    `json:"feeCharge"`
}

// ToComplainResponse converts Complain model to ComplainResponse
func (c *Complain) ToComplainResponse() *ComplainResponse {
	// Convert Complain Product Details
	productDetailsResponse := make([]ComplainProductDetailResponse, len(c.ComplainProductDetails))
	for i, productDetail := range c.ComplainProductDetails {
		productDetailResponse := ComplainProductDetailResponse{
			ProductSKU: productDetail.ProductSKU,
			Quantity:   productDetail.Quantity,
			Price:      productDetail.Price,
		}

		// Include Product data if loaded
		if productDetail.Product != nil && productDetail.Product.SKU != "" {
			productResponse := productDetail.Product.ToResponse()
			productDetailResponse.Product = productResponse
		}
		productDetailsResponse[i] = productDetailResponse
	}

	// Convert Complain User Details
	userDetailsResponse := make([]ComplainUserDetailResponse, len(c.ComplainUserDetails))
	for i, userDetail := range c.ComplainUserDetails {
		var userName string
		if userDetail.User != nil {
			userName = userDetail.User.FullName
		}

		userDetailResponse := ComplainUserDetailResponse{
			ID:        userDetail.ID,
			User:      userName,
			FeeCharge: userDetail.FeeCharge,
		}
		userDetailsResponse[i] = userDetailResponse
	}

	// Channel visual handler
	var channelName string
	if c.Channel != nil {
		channelName = c.Channel.ChannelName
	}

	// Store visual handler
	var storeName string
	if c.Store != nil {
		storeName = c.Store.StoreName
	}

	// User visual handler
	var createuser string
	if c.CreateUser != nil {
		createuser = c.CreateUser.FullName
	}

	return &ComplainResponse{
		ID:             c.ID,
		Code:           c.Code,
		TrackingNumber: c.TrackingNumber,
		OrderGineeID:   c.OrderGineeID,
		Channel:        channelName,
		Store:          storeName,
		Reason:         c.Reason,
		CreatedBy:      createuser,
		Solution:       c.Solution,
		TotalFee:       c.TotalFee,
		Checked:        c.Checked,
		CreatedAt:      c.CreatedAt.Format("02-01-2006 15:04:05"),
		UpdatedAt:      c.UpdatedAt.Format("02-01-2006 15:04:05"),
		ProductDetails: productDetailsResponse,
		UserDetails:    userDetailsResponse,
	}
}
