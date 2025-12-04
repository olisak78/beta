package models

// Plugin represents a plugin in the developer portal
type Plugin struct {
	BaseModel
	Icon               string `json:"icon" gorm:"not null;size:50" validate:"required,min=3,max=50"`
	ReactComponentPath string `json:"react_component_path" gorm:"size:500;not null" validate:"required,max=500"`
	BackendServerURL   string `json:"backend_server_url" gorm:"size:500;not null" validate:"required,max=500"`
	Owner              string `json:"owner" gorm:"size:100" validate:"max=100"`
}

// TableName returns the table name for Plugin
func (Plugin) TableName() string {
	return "plugins"
}
