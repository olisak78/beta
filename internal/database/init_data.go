package database

import (
	"developer-portal-backend/internal/database/models"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

// InitDataFromYAMLs migrate data from yaml files into db tables base on logic:
// check if row exists (predefined index - not UUID based) and whether it was changed,
// then decide whether to insert, update or do noting
func InitDataFromYAMLs(db *gorm.DB, dataDir string) error {
	if err := handleOrganizationsFromYAML(db, dataDir); err != nil {
		return err
	}
	if err := handleGroupsFromYAML(db, dataDir); err != nil {
		return err
	}
	if err := handleTeamsFromYAML(db, dataDir); err != nil {
		return err
	}
	if err := handleUsersFromYAML(db, dataDir); err != nil {
		return err
	}
	if err := handleProjectsFromYAML(db, dataDir); err != nil {
		return err
	}
	if err := handleLandscapesFromYAML(db, dataDir); err != nil {
		return err
	}
	if err := handleComponentsFromYAML(db, dataDir); err != nil {
		return err
	}
	if err := handleCategoriesFromYAML(db, dataDir); err != nil {
		return err
	}
	if err := handleLinksFromYAML(db, dataDir); err != nil {
		return err
	}
	if err := handlePluginsFromYAML(db, dataDir); err != nil {
		return err
	}
	return nil
}

func decodeOrganizations(b []byte) ([]OrganizationData, error) {
	var file OrganizationsFile
	if err := yaml.Unmarshal(b, &file); err != nil {
		return nil, err
	}
	return file.Organizations, nil
}

func decodeGroups(b []byte) ([]GroupData, error) {
	var file GroupsFile
	if err := yaml.Unmarshal(b, &file); err != nil {
		return nil, err
	}
	return file.Groups, nil
}

func decodeTeams(b []byte) ([]TeamData, error) {
	var file TeamsFile
	if err := yaml.Unmarshal(b, &file); err != nil {
		return nil, err
	}
	return file.Teams, nil
}

func decodeUsers(b []byte) ([]UserData, error) {
	var file UsersFile
	if err := yaml.Unmarshal(b, &file); err != nil {
		return nil, err
	}
	return file.Users, nil
}

func decodeProjects(b []byte) ([]ProjectData, error) {
	var file ProjectsFile
	if err := yaml.Unmarshal(b, &file); err != nil {
		return nil, err
	}
	return file.Projects, nil
}

func decodeLandscapes(b []byte) ([]LandscapeData, error) {
	var file LandscapesFile
	if err := yaml.Unmarshal(b, &file); err != nil {
		return nil, err
	}
	return file.Landscapes, nil
}

func decodeComponents(b []byte) ([]ComponentData, error) {
	var file ComponentsFile
	if err := yaml.Unmarshal(b, &file); err != nil {
		return nil, err
	}
	return file.Components, nil
}

func decodeCategories(b []byte) ([]CategoryData, error) {
	var file CategoriesFile
	if err := yaml.Unmarshal(b, &file); err != nil {
		return nil, err
	}
	return file.Categories, nil
}

func decodeLinks(b []byte) ([]LinkData, error) {
	var file LinksFile
	if err := yaml.Unmarshal(b, &file); err != nil {
		return nil, err
	}
	return file.Links, nil
}

func decodePlugins(b []byte) ([]PluginData, error) {
	var file PluginsFile
	if err := yaml.Unmarshal(b, &file); err != nil {
		return nil, err
	}
	return file.Plugins, nil
}

func handleOrganizationsFromYAML(db *gorm.DB, dataDir string) error {
	// Incremental upsert: insert new organizations, update existing if any mutable field changed.
	items, err := loadFromYAMLFile[OrganizationData](dataDir, "organizations.yaml", decodeOrganizations)
	if err != nil {
		return fmt.Errorf("load Organizations from YAML: %w", err)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		created := 0
		updated := 0
		unchanged := 0

		for _, o := range items {
			// Lookup by unique key (name) without triggering ErrRecordNotFound logs
			var org models.Organization
			lookup := tx.Where("name = ?", o.Name).Limit(1).Find(&org)
			if lookup.Error != nil {
				return fmt.Errorf("query organization %s: %w", o.Name, lookup.Error)
			}

			if lookup.RowsAffected == 0 {
				// Create new organization
				metadataJSON, err := json.Marshal(o.Metadata)
				if err != nil {
					return fmt.Errorf("marshal organization %s metadata: %w", o.Name, err)
				}
				newOrg := models.Organization{
					BaseModel: models.BaseModel{
						Name:        o.Name,
						Title:       o.Title,
						Description: o.Description,
						CreatedBy:   "cis.devops",
						Metadata:    metadataJSON,
					},
					Owner: o.Owner,
					Email: o.Email,
				}
				if err := tx.Create(&newOrg).Error; err != nil {
					return fmt.Errorf("failed to create organization: %w", err)
				}
				created++
				continue
			}

			// Existing: check if data is identical (including metadata; treat nil YAML metadata as no change)
			yamlMD, err := json.Marshal(o.Metadata)
			if err != nil {
				return fmt.Errorf("marshal organization %s metadata: %w", o.Name, err)
			}
			// Merge DB metadata with YAML overrides (YAML wins on conflicts) for comparison
			var mergedMD []byte
			if o.Metadata != nil {
				if md, err := mergeJSON(org.Metadata, yamlMD); err == nil {
					mergedMD = md
				} else {
					// Fallback to YAML if merge fails
					mergedMD = yamlMD
				}
			}
			identical := org.Title == o.Title &&
				org.Description == o.Description &&
				org.Owner == o.Owner &&
				org.Email == o.Email &&
				(o.Metadata == nil || jsonEqual(org.Metadata, mergedMD))

			if identical {
				unchanged++
				continue // no-op
			}

			// Update mutable fields (do not touch id, created_at, created_by)
			updates := map[string]interface{}{
				"title":       o.Title,
				"description": o.Description,
				"owner":       o.Owner,
				"email":       o.Email,
				"updated_by":  "cis.devops",
				"updated_at":  gorm.Expr("CURRENT_TIMESTAMP"),
			}
			// Update metadata only if provided in YAML (merge DB + YAML where YAML overrides)
			if o.Metadata != nil {
				if mergedMD != nil {
					updates["metadata"] = mergedMD
				} else {
					updates["metadata"] = yamlMD
				}
			}

			if err := tx.Model(&org).Updates(updates).Error; err != nil {
				return fmt.Errorf("update organization %s: %w", o.Name, err)
			}
			updated++
		}

		log.Printf("Organizations handling completed. %d created, %d updated, %d unchanged, %d total in YAML", created, updated, unchanged, len(items))
		return nil
	})
}

func handleGroupsFromYAML(db *gorm.DB, dataDir string) error {
	items, err := loadFromYAMLFile[GroupData](dataDir, "groups.yaml", decodeGroups)
	if err != nil {
		return fmt.Errorf("load Groups from YAML: %w", err)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		created := 0
		updated := 0
		unchanged := 0

		for _, g := range items {
			// Resolve organization by name (without ErrRecordNotFound logs)
			var org models.Organization
			orgTx := tx.Where("name = ?", g.OrgName).Limit(1).Find(&org)
			if orgTx.Error != nil {
				return fmt.Errorf("query organization %s for group %s: %w", g.OrgName, g.Name, orgTx.Error)
			}
			if orgTx.RowsAffected == 0 {
				return fmt.Errorf("organization %s not found for group %s", g.OrgName, g.Name)
			}

			// Find existing group by unique key (name, org_id)
			var group models.Group
			grpTx := tx.Where("name = ? AND org_id = ?", g.Name, org.ID).Limit(1).Find(&group)
			if grpTx.Error != nil {
				return fmt.Errorf("query group %s: %w", g.Name, grpTx.Error)
			}
			if grpTx.RowsAffected == 0 {
				// Create new group
				metadataJSON, err := json.Marshal(g.Metadata)
				if err != nil {
					return fmt.Errorf("marshal group %s metadata: %w", g.Name, err)
				}
				newGroup := models.Group{
					BaseModel: models.BaseModel{
						Name:        g.Name,
						Title:       g.Title,
						Description: g.Description,
						CreatedBy:   "cis.devops",
						Metadata:    metadataJSON,
					},
					OrgID:      org.ID,
					Owner:      g.Owner,
					Email:      g.Email,
					PictureURL: g.Picture,
				}
				if err := tx.Create(&newGroup).Error; err != nil {
					return fmt.Errorf("failed to create: %w", err)
				}
				created++
				continue
			}

			// Existing: check if data is identical (including metadata; treat nil YAML metadata as no change)
			yamlMD, err := json.Marshal(g.Metadata)
			if err != nil {
				return fmt.Errorf("marshal group %s metadata: %w", g.Name, err)
			}
			// Merge DB metadata with YAML overrides (YAML wins on conflicts) for comparison
			var mergedMD []byte
			if g.Metadata != nil {
				if md, err := mergeJSON(group.Metadata, yamlMD); err == nil {
					mergedMD = md
				} else {
					// Fallback to YAML if merge fails
					mergedMD = yamlMD
				}
			}
			identical := group.Title == g.Title &&
				group.Description == g.Description &&
				group.Owner == g.Owner &&
				group.Email == g.Email &&
				group.PictureURL == g.Picture &&
				(g.Metadata == nil || jsonEqual(group.Metadata, mergedMD))

			if identical {
				unchanged++
				continue // no-op
			}

			// Update mutable fields (do not touch id, created_at, created_by)
			updates := map[string]interface{}{
				"title":       g.Title,
				"description": g.Description,
				"owner":       g.Owner,
				"email":       g.Email,
				"picture_url": g.Picture,
				"updated_by":  "cis.devops",
				"updated_at":  gorm.Expr("CURRENT_TIMESTAMP"),
			}
			// Update metadata only if provided in YAML (merge DB + YAML where YAML overrides)
			if g.Metadata != nil {
				if mergedMD != nil {
					updates["metadata"] = mergedMD
				} else {
					updates["metadata"] = yamlMD
				}
			}

			if err := tx.Model(&group).Updates(updates).Error; err != nil {
				return fmt.Errorf("update group %s: %w", g.Name, err)
			}
			updated++
		}

		log.Printf("Groups handling completed. %d created, %d updated, %d unchanged, %d total in YAML", created, updated, unchanged, len(items))
		return nil
	})
}

func handleTeamsFromYAML(db *gorm.DB, dataDir string) error {
	items, err := loadFromYAMLFile[TeamData](dataDir, "teams.yaml", decodeTeams)
	if err != nil {
		return fmt.Errorf("load Teams from YAML: %w", err)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		created := 0
		updated := 0
		unchanged := 0

		for _, t := range items {
			// Resolve group by name without ErrRecordNotFound logs
			var group models.Group
			grpTx := tx.Where("name = ?", t.GroupName).Limit(1).Find(&group)
			if grpTx.Error != nil {
				return fmt.Errorf("query group %s for team %s: %w", t.GroupName, t.Name, grpTx.Error)
			}
			if grpTx.RowsAffected == 0 {
				return fmt.Errorf("group %s not found for team %s", t.GroupName, t.Name)
			}
			// Try find existing team by unique key (name, group_id) without ErrRecordNotFound logs
			var team models.Team
			teamTx := tx.Where("name = ? AND group_id = ?", t.Name, group.ID).Limit(1).Find(&team)
			if teamTx.Error != nil {
				return fmt.Errorf("query team %s: %w", t.Name, teamTx.Error)
			}
			if teamTx.RowsAffected == 0 {
				// Create new team
				metadataJSON, err := json.Marshal(t.Metadata)
				if err != nil {
					return fmt.Errorf("marshal team %s metadata: %w", t.Name, err)
				}
				newTeam := models.Team{
					BaseModel: models.BaseModel{
						Name:        t.Name,
						Title:       t.Title,
						Description: t.Description,
						CreatedBy:   "cis.devops",
						Metadata:    metadataJSON,
					},
					GroupID:    group.ID,
					Owner:      t.Owner,
					Email:      t.Email,
					PictureURL: t.Picture,
				}
				if err := tx.Create(&newTeam).Error; err != nil {
					return fmt.Errorf("failed to create team: %w", err)
				}
				created++
				continue
			}

			// Existing: check if data is identical (including metadata; treat nil YAML metadata as no change)
			yamlMD, err := json.Marshal(t.Metadata)
			if err != nil {
				return fmt.Errorf("marshal team %s metadata: %w", t.Name, err)
			}
			// Merge DB metadata with YAML overrides (YAML wins on conflicts) for comparison
			var mergedMD []byte
			if t.Metadata != nil {
				if md, err := mergeJSON(team.Metadata, yamlMD); err == nil {
					mergedMD = md
				} else {
					// Fallback to YAML if merge fails
					mergedMD = yamlMD
				}
			}

			identical := team.Title == t.Title &&
				team.Description == t.Description &&
				team.Owner == t.Owner &&
				team.Email == t.Email &&
				team.PictureURL == t.Picture &&
				(t.Metadata == nil || jsonEqual(team.Metadata, mergedMD))

			if identical {
				unchanged++
				continue // no-op
			}

			// Update mutable fields (do not touch id, created_at, created_by)
			updates := map[string]interface{}{
				"title":       t.Title,
				"description": t.Description,
				"owner":       t.Owner,
				"email":       t.Email,
				"picture_url": t.Picture,
				"updated_by":  "cis.devops",
				"updated_at":  gorm.Expr("CURRENT_TIMESTAMP"),
			}
			// Update metadata if provided in YAML (merge DB + YAML where YAML overrides)
			if t.Metadata != nil {
				if mergedMD != nil {
					updates["metadata"] = mergedMD
				} else {
					updates["metadata"] = yamlMD
				}
			}

			if err := tx.Model(&team).Updates(updates).Error; err != nil {
				return fmt.Errorf("update team %s: %w", t.Name, err)
			}
			updated++
		}

		log.Printf("Teams handling completed. %d created, %d updated, %d unchanged, %d total in YAML", created, updated, unchanged, len(items))
		return nil
	})
}

func handleUsersFromYAML(db *gorm.DB, dataDir string) error {
	items, err := loadFromYAMLFile[UserData](dataDir, "users.yaml", decodeUsers)
	if err != nil {
		return fmt.Errorf("load Users from YAML: %w", err)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		created := 0
		updated := 0
		unchanged := 0

		for _, u := range items {
			var dbUser models.User
			// Resolve user by name without ErrRecordNotFound logs:
			if err := tx.Where("user_id = ?", u.UserID).Limit(1).Find(&dbUser).Error; err != nil {
				return fmt.Errorf("query user %s: %w", u.UserID, err)
			}
			// if dbUser found, compare metadata.ai_instances
			if dbUser.ID != uuid.Nil {
				// Safely unmarshal existing metadata; empty/invalid, start with empty map
				var dbUserMetadata map[string]interface{}
				if len(dbUser.Metadata) > 0 {
					if err := json.Unmarshal(dbUser.Metadata, &dbUserMetadata); err != nil || dbUserMetadata == nil {
						dbUserMetadata = map[string]interface{}{}
					}
				} else {
					dbUserMetadata = map[string]interface{}{}
				}

				// YAML is the source of truth for ai_instances:
				// - If YAML provides ai_instances: set DB to that value (update only if different)
				// - If YAML omits ai_instances (including nil metadata): remove ai_instances from DB metadata if present
				var yamlAI interface{}
				yamlHasAI := false
				if u.Metadata != nil {
					if v, ok := u.Metadata["ai_instances"]; ok {
						yamlHasAI = true
						yamlAI = v
					}
				}

				existingAI, exists := dbUserMetadata["ai_instances"]

				if yamlHasAI {
					// Compare existing vs YAML value semantically
					existingJSONBytes, err := json.Marshal(existingAI)
					if err != nil {
						return fmt.Errorf("marshal existing ai_instances for user %s: %w", u.UserID, err)
					}
					yamlJSONBytes, err := json.Marshal(yamlAI)
					if err != nil {
						return fmt.Errorf("marshal YAML ai_instances for user %s: %w", u.UserID, err)
					}

					if !jsonEqual(json.RawMessage(existingJSONBytes), json.RawMessage(yamlJSONBytes)) {
						// Update metadata.ai_instances to YAML-provided value
						dbUserMetadata["ai_instances"] = yamlAI
						updatedMetadataJSON, err := json.Marshal(dbUserMetadata)
						if err != nil {
							return fmt.Errorf("marshal updated metadata for user %s: %w", u.UserID, err)
						}
						if err := tx.Model(&dbUser).Updates(map[string]interface{}{
							"metadata":   updatedMetadataJSON,
							"updated_by": "cis.devops",
							"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
						}).Error; err != nil {
							return fmt.Errorf("update user %s metadata.ai_instances: %w", u.UserID, err)
						}
						updated++
						continue
					}

					unchanged++
					continue
				} else {
					// YAML omits ai_instances: ensure DB does not have it
					if exists {
						delete(dbUserMetadata, "ai_instances")
						updatedMetadataJSON, err := json.Marshal(dbUserMetadata)
						if err != nil {
							return fmt.Errorf("marshal updated metadata for user %s: %w", u.UserID, err)
						}
						if err := tx.Model(&dbUser).Updates(map[string]interface{}{
							"metadata":   updatedMetadataJSON,
							"updated_by": "cis.devops",
							"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
						}).Error; err != nil {
							return fmt.Errorf("remove user %s metadata.ai_instances: %w", u.UserID, err)
						}
						updated++
						continue
					}

					unchanged++
					continue
				}
			}
			// Create new user
			metadataJSON, err := json.Marshal(u.Metadata)
			if err != nil {
				return fmt.Errorf("marshal user %s metadata: %w", u.UserID, err)
			}
			fullTitle := u.FirstName + " " + u.LastName
			newUser := models.User{
				BaseModel: models.BaseModel{
					Name:        u.UserID,
					Title:       fullTitle,
					Description: "",
					CreatedBy:   "cis.devops",
					Metadata:    metadataJSON,
				},
				UserID:     u.UserID,
				FirstName:  u.FirstName,
				LastName:   u.LastName,
				Email:      u.Email,
				Mobile:     u.PhoneNumber,
				TeamDomain: models.TeamDomain(u.TeamDomain),
				TeamRole:   models.TeamRole(u.TeamRole),
				Metadata:   metadataJSON,
			}
			// Optionally resolve team by name for assignment (do not enforce group/org existence)
			var team models.Team
			if u.TeamName != "" {
				if err := tx.Where("name = ?", u.TeamName).Limit(1).Find(&team).Error; err != nil {
					return fmt.Errorf("query team %s for user %s: %w", u.TeamName, u.UserID, err)
				}
			}
			if team.ID != uuid.Nil {
				newUser.TeamID = &team.ID
			}

			if err := tx.Create(&newUser).Error; err != nil {
				return fmt.Errorf("failed to create user %s: %w", u.UserID, err)
			}
			created++
		}
		log.Printf("Users handling completed. %d created, %d updated, %d unchanged, %d total in YAML", created, updated, unchanged, len(items))
		return nil

	})
}

func handleProjectsFromYAML(db *gorm.DB, dataDir string) error {
	items, err := loadFromYAMLFile[ProjectData](dataDir, "projects.yaml", decodeProjects)
	if err != nil {
		return fmt.Errorf("load Projects from YAML: %w", err)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		created := 0
		updated := 0
		unchanged := 0

		for _, p := range items {
			// Lookup by unique key (name)
			var proj models.Project
			projTx := tx.Where("name = ?", p.Name).Limit(1).Find(&proj)
			if projTx.Error != nil {
				return fmt.Errorf("query project %s: %w", p.Name, projTx.Error)
			}

			if projTx.RowsAffected == 0 {
				// Create new project
				metadataJSON, err := json.Marshal(p.Metadata)
				if err != nil {
					return fmt.Errorf("marshal project %s metadata: %w", p.Name, err)
				}
				newProj := models.Project{
					BaseModel: models.BaseModel{
						Name:        p.Name,
						Title:       p.Title,
						Description: p.Description,
						CreatedBy:   "cis.devops",
						Metadata:    metadataJSON,
					},
				}
				if err := tx.Create(&newProj).Error; err != nil {
					return fmt.Errorf("failed to create project: %w", err)
				}
				created++
				continue
			}

			// Existing: check if data is identical (including metadata; treat nil YAML metadata as no change)
			yamlMD, err := json.Marshal(p.Metadata)
			if err != nil {
				return fmt.Errorf("marshal project %s metadata: %w", p.Name, err)
			}
			// Merge DB metadata with YAML overrides (YAML wins on conflicts) for comparison
			var mergedMD []byte
			if p.Metadata != nil {
				if md, err := mergeJSON(proj.Metadata, yamlMD); err == nil {
					mergedMD = md
				} else {
					// Fallback to YAML if merge fails
					mergedMD = yamlMD
				}
			}
			identical := proj.Title == p.Title &&
				proj.Description == p.Description &&
				(p.Metadata == nil || jsonEqual(proj.Metadata, mergedMD))

			if identical {
				unchanged++
				continue // no-op
			}

			// Update mutable fields (do not touch id, created_at, created_by)
			updates := map[string]interface{}{
				"title":       p.Title,
				"description": p.Description,
				"updated_by":  "cis.devops",
				"updated_at":  gorm.Expr("CURRENT_TIMESTAMP"),
			}
			// Update metadata only if provided in YAML (merge DB + YAML where YAML overrides)
			if p.Metadata != nil {
				if mergedMD != nil {
					updates["metadata"] = mergedMD
				} else {
					updates["metadata"] = yamlMD
				}
			}

			if err := tx.Model(&proj).Updates(updates).Error; err != nil {
				return fmt.Errorf("update project %s: %w", p.Name, err)
			}
			updated++
		}

		log.Printf("Projects handling completed. %d created, %d updated, %d unchanged, %d total in YAML", created, updated, unchanged, len(items))
		return nil
	})
}

func handleLandscapesFromYAML(db *gorm.DB, dataDir string) error {
	items, err := loadFromYAMLFile[LandscapeData](dataDir, "landscapes.yaml", decodeLandscapes)
	if err != nil {
		return fmt.Errorf("load Landscapes from YAML: %w", err)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		created := 0
		updated := 0
		unchanged := 0

		for _, l := range items {
			// Resolve project by name (without ErrRecordNotFound logs)
			var proj models.Project
			projTx := tx.Where("name = ?", l.Project).Limit(1).Find(&proj)
			if projTx.Error != nil {
				return fmt.Errorf("query project %s for landscape %s: %w", l.Project, l.Name, projTx.Error)
			}
			if projTx.RowsAffected == 0 {
				return fmt.Errorf("project %s not found for landscape %s", l.Project, l.Name)
			}

			env := strings.ToLower(l.Environment)

			// Find existing landscape by unique key (name, project_id)
			var landscape models.Landscape
			lsTx := tx.Where("name = ? AND project_id = ?", l.Name, proj.ID).Limit(1).Find(&landscape)
			if lsTx.Error != nil {
				return fmt.Errorf("query landscape %s: %w", l.Name, lsTx.Error)
			}
			if lsTx.RowsAffected == 0 {
				// Create new landscape
				metadataJSON, err := json.Marshal(l.Metadata)
				if err != nil {
					return fmt.Errorf("marshal landscape %s metadata: %w", l.Name, err)
				}
				newLandscape := models.Landscape{
					BaseModel: models.BaseModel{
						Name:        l.Name,
						Title:       l.Title,
						Description: l.Description,
						CreatedBy:   "cis.devops",
						Metadata:    metadataJSON,
					},
					ProjectID:   proj.ID,
					Domain:      l.Domain,
					Environment: env,
				}
				if err := tx.Create(&newLandscape).Error; err != nil {
					return fmt.Errorf("failed to create landscape: %w", err)
				}
				created++
				continue
			}

			// Existing: check if data is identical (including metadata; treat nil YAML metadata as no change)
			yamlMD, err := json.Marshal(l.Metadata)
			if err != nil {
				return fmt.Errorf("marshal landscape %s metadata: %w", l.Name, err)
			}
			// Merge DB metadata with YAML overrides (YAML wins on conflicts) for comparison
			var mergedMD []byte
			if l.Metadata != nil {
				if md, err := mergeJSON(landscape.Metadata, yamlMD); err == nil {
					mergedMD = md
				} else {
					// Fallback to YAML if merge fails
					mergedMD = yamlMD
				}
			}
			identical := landscape.Title == l.Title &&
				landscape.Description == l.Description &&
				landscape.Domain == l.Domain &&
				landscape.Environment == env &&
				(l.Metadata == nil || jsonEqual(landscape.Metadata, mergedMD))

			if identical {
				unchanged++
				continue // no-op
			}

			// Update mutable fields (do not touch id, created_at, created_by, project_id)
			updates := map[string]interface{}{
				"title":       l.Title,
				"description": l.Description,
				"domain":      l.Domain,
				"environment": env,
				"updated_by":  "cis.devops",
				"updated_at":  gorm.Expr("CURRENT_TIMESTAMP"),
			}
			// Update metadata only if provided in YAML (merge DB + YAML where YAML overrides)
			if l.Metadata != nil {
				if mergedMD != nil {
					updates["metadata"] = mergedMD
				} else {
					updates["metadata"] = yamlMD
				}
			}

			if err := tx.Model(&landscape).Updates(updates).Error; err != nil {
				return fmt.Errorf("update landscape %s: %w", l.Name, err)
			}
			updated++
		}

		log.Printf("Landscapes handling completed. %d created, %d updated, %d unchanged, %d total in YAML", created, updated, unchanged, len(items))
		return nil
	})
}

func handleComponentsFromYAML(db *gorm.DB, dataDir string) error {
	items, err := loadFromYAMLFile[ComponentData](dataDir, "components.yaml", decodeComponents)
	if err != nil {
		return fmt.Errorf("load Components from YAML: %w", err)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		created := 0
		updated := 0
		unchanged := 0

		for _, c := range items {
			// Resolve project by name (without ErrRecordNotFound logs)
			var proj models.Project
			projTx := tx.Where("name = ?", c.Project).Limit(1).Find(&proj)
			if projTx.Error != nil {
				return fmt.Errorf("query project %s for component %s: %w", c.Project, c.Name, projTx.Error)
			}
			if projTx.RowsAffected == 0 {
				return fmt.Errorf("project %s not found for component %s", c.Project, c.Name)
			}

			// Resolve owner team by name
			var team models.Team
			teamTx := tx.Where("name = ?", c.Owner).Limit(1).Find(&team)
			if teamTx.Error != nil {
				return fmt.Errorf("query owner team %s for component %s: %w", c.Owner, c.Name, teamTx.Error)
			}
			if teamTx.RowsAffected == 0 {
				return fmt.Errorf("owner team %s not found for component %s", c.Owner, c.Name)
			}

			// Find existing component by unique key (name, project_id)
			var component models.Component
			compTx := tx.Where("name = ? AND project_id = ?", c.Name, proj.ID).Limit(1).Find(&component)
			if compTx.Error != nil {
				return fmt.Errorf("query component %s: %w", c.Name, compTx.Error)
			}
			if compTx.RowsAffected == 0 {
				// Create new component
				metadataJSON, err := json.Marshal(c.Metadata)
				if err != nil {
					return fmt.Errorf("marshal component %s metadata: %w", c.Name, err)
				}
				newComponent := models.Component{
					BaseModel: models.BaseModel{
						Name:        c.Name,
						Title:       c.Title,
						Description: c.Description,
						CreatedBy:   "cis.devops",
						Metadata:    metadataJSON,
					},
					ProjectID: proj.ID,
					OwnerID:   team.ID,
				}
				if err := tx.Create(&newComponent).Error; err != nil {
					return fmt.Errorf("failed to create component: %w", err)
				}
				created++
				continue
			}

			// Existing check if data is identical (including metadata; treat nil YAML metadata as no change)
			yamlMD, err := json.Marshal(c.Metadata)
			if err != nil {
				return fmt.Errorf("marshal component %s metadata: %w", c.Name, err)
			}
			// Merge DB metadata with YAML overrides (YAML wins on conflicts) for comparison
			var mergedMD []byte
			if c.Metadata != nil {
				if md, err := mergeJSON(component.Metadata, yamlMD); err == nil {
					mergedMD = md
				} else {
					// Fallback to YAML if merge fails
					mergedMD = yamlMD
				}
			}
			identical := component.Title == c.Title &&
				component.Description == c.Description &&
				component.OwnerID == team.ID &&
				(c.Metadata == nil || jsonEqual(component.Metadata, mergedMD))

			if identical {
				unchanged++
				continue // no-op
			}

			// Update mutable fields (do not touch id, created_at, created_by, project_id)
			updates := map[string]interface{}{
				"title":       c.Title,
				"description": c.Description,
				"owner_id":    team.ID,
				"updated_by":  "cis.devops",
				"updated_at":  gorm.Expr("CURRENT_TIMESTAMP"),
			}
			// Update metadata only if provided in YAML (merge DB + YAML where YAML overrides)
			if c.Metadata != nil {
				if mergedMD != nil {
					updates["metadata"] = mergedMD
				} else {
					updates["metadata"] = yamlMD
				}
			}

			if err := tx.Model(&component).Updates(updates).Error; err != nil {
				return fmt.Errorf("update component %s: %w", c.Name, err)
			}
			updated++
		}

		log.Printf("Components handling completed. %d created, %d updated, %d unchanged, %d total in YAML", created, updated, unchanged, len(items))
		return nil
	})
}

func handleCategoriesFromYAML(db *gorm.DB, dataDir string) error {
	items, err := loadFromYAMLFile[CategoryData](dataDir, "categories.yaml", decodeCategories)
	if err != nil {
		return fmt.Errorf("load Categories from YAML: %w", err)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		created := 0
		updated := 0
		unchanged := 0

		for _, c := range items {
			// Lookup by unique key (name)
			var cat models.Category
			catTx := tx.Where("name = ?", c.Name).Limit(1).Find(&cat)
			if catTx.Error != nil {
				return fmt.Errorf("query category %s: %w", c.Name, catTx.Error)
			}

			if catTx.RowsAffected == 0 {
				// Create new category
				metadataJSON, err := json.Marshal(c.Metadata)
				if err != nil {
					return fmt.Errorf("marshal category %s metadata: %w", c.Name, err)
				}
				newCat := models.Category{
					BaseModel: models.BaseModel{
						Name:        c.Name,
						Title:       c.Title,
						Description: c.Description,
						CreatedBy:   "cis.devops",
						Metadata:    metadataJSON,
					},
					Icon:  c.Icon,
					Color: c.Color,
				}
				if err := tx.Create(&newCat).Error; err != nil {
					return fmt.Errorf("failed to create category: %w", err)
				}
				created++
				continue
			}

			// Existing: check if data is identical (including metadata; treat nil YAML metadata as no change)
			yamlMD, err := json.Marshal(c.Metadata)
			if err != nil {
				return fmt.Errorf("marshal category %s metadata: %w", c.Name, err)
			}
			// Merge DB metadata with YAML overrides (YAML wins on conflicts) for comparison
			var mergedMD []byte
			if c.Metadata != nil {
				if md, err := mergeJSON(cat.Metadata, yamlMD); err == nil {
					mergedMD = md
				} else {
					// Fallback to YAML if merge fails
					mergedMD = yamlMD
				}
			}
			identical := cat.Title == c.Title &&
				cat.Description == c.Description &&
				cat.Icon == c.Icon &&
				cat.Color == c.Color &&
				(c.Metadata == nil || jsonEqual(cat.Metadata, mergedMD))

			if identical {
				unchanged++
				continue // no-op
			}

			// Update mutable fields (do not touch id, created_at, created_by)
			updates := map[string]interface{}{
				"title":       c.Title,
				"description": c.Description,
				"icon":        c.Icon,
				"color":       c.Color,
				"updated_by":  "cis.devops",
				"updated_at":  gorm.Expr("CURRENT_TIMESTAMP"),
			}
			// Update metadata only if provided in YAML (merge DB + YAML where YAML overrides)
			if c.Metadata != nil {
				if mergedMD != nil {
					updates["metadata"] = mergedMD
				} else {
					updates["metadata"] = yamlMD
				}
			}

			if err := tx.Model(&cat).Updates(updates).Error; err != nil {
				return fmt.Errorf("update category %s: %w", c.Name, err)
			}
			updated++
		}

		log.Printf("Categories handling completed. %d created, %d updated, %d unchanged, %d total in YAML", created, updated, unchanged, len(items))
		return nil
	})
}

func handleLinksFromYAML(db *gorm.DB, dataDir string) error {
	items, err := loadFromYAMLFile[LinkData](dataDir, "links.yaml", decodeLinks)
	if err != nil {
		return fmt.Errorf("load Links from YAML: %w", err)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		created := 0
		updated := 0
		unchanged := 0

		// Resolve owner user by fixed user_id 'cis.devops'
		var owner models.User
		ownerTx := tx.Where("user_id = ?", "cis.devops").Limit(1).Find(&owner)
		if ownerTx.Error != nil {
			return fmt.Errorf("query owner user 'cis.devops': %w", ownerTx.Error)
		}
		if ownerTx.RowsAffected == 0 {
			log.Printf("⚠️  Links handling skipped: owner user 'cis.devops' not found")
			return nil
		}

		for _, l := range items {
			// Resolve category by name
			var cat models.Category
			catTx := tx.Where("name = ?", l.Category).Limit(1).Find(&cat)
			if catTx.Error != nil {
				return fmt.Errorf("query category %s for link %s: %w", l.Category, l.Title, catTx.Error)
			}
			if catTx.RowsAffected == 0 {
				return fmt.Errorf("category %s not found for link %s", l.Category, l.Title)
			}

			// Derive stable name from title
			name := slugifyTitle(l.Title)

			// Normalize tags to CSV string
			tagsCSV := normalizeTagsCSV(l.TagsRaw)

			// Find existing link by (name, category_id, created_by='cis.devops') only
			var link models.Link
			lookup := tx.Where("name = ? AND category_id = ? AND created_by = ?", name, cat.ID, "cis.devops").Limit(1).Find(&link)
			if lookup.Error != nil {
				return fmt.Errorf("query link %s: %w", l.Title, lookup.Error)
			}

			yamlMD, err := json.Marshal(l.Metadata)
			if err != nil {
				return fmt.Errorf("marshal link %s metadata: %w", l.Title, err)
			}

			if lookup.RowsAffected == 0 {
				// Create new link
				newLink := models.Link{
					BaseModel: models.BaseModel{
						Name:        name,
						Title:       l.Title,
						Description: l.Description,
						CreatedBy:   "cis.devops",
						Metadata:    yamlMD,
					},
					Owner:      owner.ID,
					URL:        l.URL,
					CategoryID: cat.ID,
					Tags:       tagsCSV,
				}
				if err := tx.Create(&newLink).Error; err != nil {
					return fmt.Errorf("failed to create link %s: %w", l.Title, err)
				}
				created++
				continue
			}

			// Merge DB metadata with YAML overrides (YAML wins on conflicts) for comparison
			var mergedMD []byte
			if l.Metadata != nil {
				if md, err := mergeJSON(link.Metadata, yamlMD); err == nil {
					mergedMD = md
				} else {
					// Fallback to YAML if merge fails
					mergedMD = yamlMD
				}
			}
			// Compare fields: title, description, metadata, url, category_id, tags
			identical := link.Title == l.Title &&
				link.Description == l.Description &&
				link.URL == l.URL &&
				link.CategoryID == cat.ID &&
				link.Tags == tagsCSV &&
				(l.Metadata == nil || jsonEqual(link.Metadata, mergedMD))

			if identical {
				unchanged++
				continue // no-op
			}

			updatedDescription := l.Description
			if updatedDescription == "" {
				updatedDescription = link.Description
			}

			updates := map[string]interface{}{
				"title":       l.Title,
				"description": updatedDescription,
				"url":         l.URL,
				"category_id": cat.ID,
				"tags":        tagsCSV,
				"updated_by":  "cis.devops",
				"updated_at":  gorm.Expr("CURRENT_TIMESTAMP"),
			}
			if l.Metadata != nil {
				if mergedMD != nil {
					updates["metadata"] = mergedMD
				} else {
					updates["metadata"] = yamlMD
				}
			}

			if err := tx.Model(&link).Updates(updates).Error; err != nil {
				return fmt.Errorf("update link %s: %w", l.Title, err)
			}
			updated++
		}

		log.Printf("Links handling completed. %d created, %d updated, %d unchanged, %d total in YAML", created, updated, unchanged, len(items))
		return nil
	})
}

func handlePluginsFromYAML(db *gorm.DB, dataDir string) error {
	items, err := loadFromYAMLFile[PluginData](dataDir, "plugins.yaml", decodePlugins)
	if err != nil {
		return fmt.Errorf("load Plugins from YAML: %w", err)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		created := 0
		updated := 0
		unchanged := 0

		for _, p := range items {
			// Lookup by unique key (name)
			var plugin models.Plugin
			pluginTx := tx.Where("name = ?", p.Name).Limit(1).Find(&plugin)
			if pluginTx.Error != nil {
				return fmt.Errorf("query plugin %s: %w", p.Name, pluginTx.Error)
			}

			if pluginTx.RowsAffected == 0 {
				// Create new plugin
				metadataJSON, err := json.Marshal(p.Metadata)
				if err != nil {
					return fmt.Errorf("marshal plugin %s metadata: %w", p.Name, err)
				}
				newPlugin := models.Plugin{
					BaseModel: models.BaseModel{
						Name:        p.Name,
						Title:       p.Title,
						Description: p.Description,
						CreatedBy:   "cis.devops",
						Metadata:    metadataJSON,
					},
					Icon:               p.Icon,
					ReactComponentPath: p.ReactComponentPath,
					BackendServerURL:   p.BackendServerURL,
					Owner:              p.Owner,
				}
				if err := tx.Create(&newPlugin).Error; err != nil {
					return fmt.Errorf("failed to create plugin: %w", err)
				}
				created++
				continue
			}

			// Existing: check if data is identical (including metadata; treat nil YAML metadata as no change)
			yamlMD, err := json.Marshal(p.Metadata)
			if err != nil {
				return fmt.Errorf("marshal plugin %s metadata: %w", p.Name, err)
			}
			// Merge DB metadata with YAML overrides (YAML wins on conflicts) for comparison
			var mergedMD []byte
			if p.Metadata != nil {
				if md, err := mergeJSON(plugin.Metadata, yamlMD); err == nil {
					mergedMD = md
				} else {
					// Fallback to YAML if merge fails
					mergedMD = yamlMD
				}
			}
			identical := plugin.Title == p.Title &&
				plugin.Description == p.Description &&
				plugin.Icon == p.Icon &&
				plugin.ReactComponentPath == p.ReactComponentPath &&
				plugin.BackendServerURL == p.BackendServerURL &&
				plugin.Owner == p.Owner &&
				(p.Metadata == nil || jsonEqual(plugin.Metadata, mergedMD))

			if identical {
				unchanged++
				continue // no-op
			}

			// Update mutable fields (do not touch id, created_at, created_by)
			updates := map[string]interface{}{
				"title":                p.Title,
				"description":          p.Description,
				"icon":                 p.Icon,
				"react_component_path": p.ReactComponentPath,
				"backend_server_url":   p.BackendServerURL,
				"owner":                p.Owner,
				"updated_by":           "cis.devops",
				"updated_at":           gorm.Expr("CURRENT_TIMESTAMP"),
			}
			// Update metadata only if provided in YAML (merge DB + YAML where YAML overrides)
			if p.Metadata != nil {
				if mergedMD != nil {
					updates["metadata"] = mergedMD
				} else {
					updates["metadata"] = yamlMD
				}
			}

			if err := tx.Model(&plugin).Updates(updates).Error; err != nil {
				return fmt.Errorf("update plugin %s: %w", p.Name, err)
			}
			updated++
		}

		log.Printf("Plugins handling completed. %d created, %d updated, %d unchanged, %d total in YAML", created, updated, unchanged, len(items))
		return nil
	})
}
