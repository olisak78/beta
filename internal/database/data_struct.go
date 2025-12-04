package database

// OrganizationData represents organization YAML structure in organizations.yaml
type OrganizationData struct {
	Name        string                 `yaml:"name"`
	Title       string                 `yaml:"title"`
	Description string                 `yaml:"description"`
	Owner       string                 `yaml:"owner"`
	Email       string                 `yaml:"email"`
	Metadata    map[string]interface{} `yaml:"metadata,omitempty"`
}

// OrganizationsFile wraps the organizations array
type OrganizationsFile struct {
	Organizations []OrganizationData `yaml:"organizations"`
}

// GroupData represents group YAML structure in groups.yaml
type GroupData struct {
	Name        string                 `yaml:"name"`
	Title       string                 `yaml:"title"`
	Description string                 `yaml:"description"`
	Owner       string                 `yaml:"owner"`
	Email       string                 `yaml:"email"`
	OrgName     string                 `yaml:"org"`
	Picture     string                 `yaml:"picture_url"`
	Metadata    map[string]interface{} `yaml:"metadata,omitempty"`
}

// GroupsFile wraps the groups array
type GroupsFile struct {
	Groups []GroupData `yaml:"groups"`
}

// TeamData represents team YAML structure in teams.yaml
type TeamData struct {
	Name        string                 `yaml:"name"`
	GroupName   string                 `yaml:"group_name"`
	Title       string                 `yaml:"title"`
	Description string                 `yaml:"description"`
	Owner       string                 `yaml:"owner"`
	Email       string                 `yaml:"email"`
	Picture     string                 `yaml:"picture_url"`
	Metadata    map[string]interface{} `yaml:"metadata,omitempty"`
}

// TeamsFile wraps the teams array
type TeamsFile struct {
	Teams []TeamData `yaml:"teams"`
}

// UserData represents user YAML structure in users.yaml
type UserData struct {
	UserID      string                 `yaml:"name"`
	TeamName    string                 `yaml:"team_name"`
	FirstName   string                 `yaml:"first_name"`
	LastName    string                 `yaml:"last_name"`
	Email       string                 `yaml:"email"`
	PhoneNumber string                 `yaml:"phone_number"`
	TeamDomain  string                 `yaml:"team_domain"`
	TeamRole    string                 `yaml:"team_role"`
	Metadata    map[string]interface{} `yaml:"metadata,omitempty"`
}

// UsersFile wraps the users array
type UsersFile struct {
	Users []UserData `yaml:"users"`
}

// ProjectData represents project YAML structure in projects.yaml
type ProjectData struct {
	Name        string                 `yaml:"name"`
	Title       string                 `yaml:"title"`
	Description string                 `yaml:"description"`
	Metadata    map[string]interface{} `yaml:"metadata,omitempty"`
}

// ProjectsFile wraps the projects array
type ProjectsFile struct {
	Projects []ProjectData `yaml:"projects"`
}

// LandscapeData represents landscape YAML structure in landscapes.yaml
type LandscapeData struct {
	Name        string                 `yaml:"name"`
	Title       string                 `yaml:"title"`
	Description string                 `yaml:"description"`
	Project     string                 `yaml:"project"`
	Environment string                 `yaml:"environment"`
	Domain      string                 `yaml:"domain,omitempty"`
	Metadata    map[string]interface{} `yaml:"metadata,omitempty"`
}

// LandscapesFile wraps the landscapes array
type LandscapesFile struct {
	Landscapes []LandscapeData `yaml:"landscapes"`
}

// ComponentData represents component YAML structure in components.yaml
type ComponentData struct {
	Name        string                 `yaml:"name"`
	Title       string                 `yaml:"title"`
	Description string                 `yaml:"description"`
	Owner       string                 `yaml:"owner"`
	Project     string                 `yaml:"project"`
	Metadata    map[string]interface{} `yaml:"metadata,omitempty"`
}

// ComponentsFile wraps the components array
type ComponentsFile struct {
	Components []ComponentData `yaml:"components"`
}

// CategoryData represents category YAML structure in categories.yaml
type CategoryData struct {
	Name        string                 `yaml:"name"`
	Title       string                 `yaml:"title"`
	Description string                 `yaml:"description,omitempty"`
	Icon        string                 `yaml:"icon"`
	Color       string                 `yaml:"color"`
	Metadata    map[string]interface{} `yaml:"metadata,omitempty"`
}

// CategoriesFile wraps the categories array
type CategoriesFile struct {
	Categories []CategoryData `yaml:"categories"`
}

// LinkData represents link YAML structure in links.yaml
type LinkData struct {
	Title       string                 `yaml:"title"`
	Description string                 `yaml:"description"`
	URL         string                 `yaml:"url"`
	Category    string                 `yaml:"category"`
	TagsRaw     interface{}            `yaml:"tags"`
	Metadata    map[string]interface{} `yaml:"metadata,omitempty"`
}

// LinksFile wraps the links array
type LinksFile struct {
	Links []LinkData `yaml:"links"`
}

// PluginData represents plugin YAML structure in plugins.yaml
type PluginData struct {
	Name               string                 `yaml:"name"`
	Title              string                 `yaml:"title"`
	Description        string                 `yaml:"description"`
	Icon               string                 `yaml:"icon"`
	ReactComponentPath string                 `yaml:"react_component_path"`
	BackendServerURL   string                 `yaml:"backend_server_url"`
	Owner              string                 `yaml:"owner"`
	Metadata           map[string]interface{} `yaml:"metadata,omitempty"`
}

// PluginsFile wraps the plugins array
type PluginsFile struct {
	Plugins []PluginData `yaml:"plugins"`
}
