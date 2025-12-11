package handlers

import (
	"net/http"

	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/repository"
	"developer-portal-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// LDAPHandler handles LDAP-related HTTP requests
type LDAPHandler struct {
	service  service.LDAPServiceInterface
	userRepo repository.UserRepositoryInterface
}

// NewLDAPHandler creates a new LDAP handler
func NewLDAPHandler(s service.LDAPServiceInterface, r repository.UserRepositoryInterface) *LDAPHandler {
	return &LDAPHandler{service: s, userRepo: r}
}

// UserSearch searches LDAP users by name prefix
// @Summary Search LDAP users by name prefix
// @Description Searches LDAP directory for users where name starts with given prefix
// @Tags users
// @Produce json
// @Param name query string true "Name prefix"
// @Success 200 {object} service.LDAPUserSearchResponse "Search results"
// @Failure 400 {object} map[string]interface{} "Missing or invalid query parameter"
// @Failure 502 {object} map[string]interface{} "LDAP connection or search failed"
// @Security BearerAuth
// @Router /users/search/new [get]
func (h *LDAPHandler) UserSearch(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": apperrors.NewMissingQueryParam("name").Error()})
		return
	}

	// 1) Query LDAP by CN prefix
	users, ldapErr := h.service.SearchUsersByCN(name)
	if ldapErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "ldap search failed: " + ldapErr.Error()})
		return
	}

	// 2) Extract user_ids from LDAP results (LDAP 'name' corresponds to users.user_id)
	ids := make([]string, 0, len(users))
	for _, u := range users {
		if u.Name != "" {
			ids = append(ids, u.Name)
		}
	}

	// 3) Query DB for which of those user_ids already exist
	existing, repoErr := h.userRepo.GetExistingUserIDs(ids)
	if repoErr != nil {
		// If DB check fails, proceed assuming none exist
		existing = []string{}
	}

	// 4) Build set of existing user_ids
	existingSet := make(map[string]struct{}, len(existing))
	for _, id := range existing {
		if id != "" {
			existingSet[id] = struct{}{}
		}
	}

	// 5) Build response; 'new' is true if user_id not found in DB
	resp := make([]gin.H, 0, len(users))
	for _, u := range users {
		_, exists := existingSet[u.Name] // u.Name is the LDAP-provided user_id
		resp = append(resp, gin.H{
			"id":         u.Name,
			"first_name": u.GivenName,
			"last_name":  u.SN,
			"email":      u.Mail,
			"mobile":     u.Mobile,
			"new":        !exists,
		})
	}
	c.JSON(http.StatusOK, gin.H{"result": resp})
}
