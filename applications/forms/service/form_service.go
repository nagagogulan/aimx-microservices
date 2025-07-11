package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/smtp"
	"os"
	"strings"
	"time"

	errcom "github.com/PecozQ/aimx-library/apperrors"
	"github.com/PecozQ/aimx-library/common"
	commonlib "github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/entities"
	kafkas "github.com/PecozQ/aimx-library/kafka"

	middleware "github.com/PecozQ/aimx-library/middleware"
	"github.com/gofrs/uuid"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"whatsdare.com/fullstack/aimx/backend/model"
)

func (s *service) CreateForm(ctx context.Context, form dto.FormDTO) (*dto.FormDTO, error) {

	if form.Type == 1 {
		orgreq := &dto.CreateOrganizationRequest{}
		for _, val := range form.Fields {
			switch val.Label {
			case "Admin Email Address":
				if email, ok := val.Value.(string); ok {
					orgreq.Email = email
				}
			}
		}
		domainParts := strings.Split(orgreq.Email, "@")
		if len(domainParts) < 2 {
			return nil, errcom.ErrInvalidEmail
		}
		orgDomain := domainParts[1]

		formList, err := s.formRepo.GetFormAll(ctx, form.Type)
		if err != nil {
			return nil, err
		}
		for _, form := range formList {
			for _, field := range form.Fields {
				if field.Label == "Admin Email Address" {
					fmt.Println("Found Admin Email Address field:")
					fmt.Printf("ID: %d, Placeholder: %s, Value: %v\n", field.ID, field.Placeholder, field.Value)

					// Assert that field.Value is a string
					email, ok := field.Value.(string)
					if !ok {
						return nil, errcom.ErrInvalidEmail // Or fmt.Errorf("email value is not a string")
					}
					domainParts := strings.Split(email, "@")
					if len(domainParts) != 2 {
						return nil, errcom.ErrInvalidEmail
					}

					orgDomainInForm := domainParts[1]
					if orgDomain == orgDomainInForm && form.Status != 2 {
						if form.Status == 10 {
							return nil, errcom.ErrOrganizationDeactivated
						}
						return nil, errcom.ErrDomainExist
					}
				}
			}
		}

		existingOrg, err := s.organizationRepo.GetOrganizationByDomain(ctx, orgDomain)
		if err != nil {
			return nil, errcom.ErrInvalidEmail
		}
		if !commonlib.IsEmpty(existingOrg) {
			return nil, errcom.ErrDuplicateEmail
		}
		if existingOrg != nil && form.Status == 2 {
			createdForm, err := s.formRepo.CreateForm(ctx, form)
			if err != nil {
				commonlib.LogMessage(s.logger, commonlib.Error, "CreateTemplate", err.Error(), err, "CreateBy", createdForm)
				return nil, errcom.ErrUnableToCreate
			}
			return createdForm, err
		}

	} else if form.Type == 3 {
		orgID, _ := ctx.Value(middleware.CtxOrganizationIDKey).(string)
		userID, _ := ctx.Value(middleware.CtxUserIDKey).(string)
		userId, err := uuid.FromString(userID)
		if err != nil {
			return nil, errcom.ErrUserNotFound
		}
		userDetail, err := s.userRepo.GetUserByID(ctx, userId)
		ctx = context.WithValue(ctx, "role", userDetail.Role.Name)
		userrole, _ := ctx.Value("role").(string)
		fmt.Println("Role", userrole)
		if userID != "" {
			form.UserID = userID
		}
		if orgID != "" && (userrole == "SuperAdmin" || userrole == "Collaborator") {
			settings, err := s.globalSettingRepo.GetAllGeneralSetting()
			if err != nil {
				return nil, errcom.ErrRecordNotFounds
			}
			typeTotal, err := s.formRepo.CountFormsByType(ctx, form.Type)
			if err != nil {
				return nil, err
			}
			if typeTotal > 0 && settings != nil {
				if typeTotal >= int64(settings.MaxActiveProjects) {
					return nil, errcom.ErrMaxActiveProjectReaxched
				}
			}
		}
		if orgID != "" && (userrole == "Admin" || userrole == "User") {
			orgsettings, err := s.orgSettingRepo.GetOrganizationSettingByOrgID(ctx, orgID)
			if err != nil {
				return nil, errcom.ErrRecordNotFounds
			}
			typeTotal, err := s.formRepo.CountFormsByType(ctx, form.Type)
			if err != nil {
				return nil, err
			}
			if typeTotal > 0 && orgsettings != nil {
				if typeTotal >= int64(orgsettings.MaxActiveProjects) {
					return nil, errcom.ErrMaxActiveProjectReaxched
				}
			}
		}
		// Validate metadata for type=3 (docket form)
		fmt.Printf("Form: %+v\n", form)

		// Look for metadata in the Fields array
		var metadataField map[string]interface{}

		// First check if form.MetaData is already set
		if form.MetaData != nil {
			metadata, ok := form.MetaData.(map[string]interface{})
			if ok && metadata != nil {
				// Use existing metadata
				metadataField = metadata
			}
		}

		// If metadata is not set, try to find it in the Fields
		if metadataField == nil {
			for _, field := range form.Fields {
				if field.Label == "MetaData" {
					if metaValue, ok := field.Value.(map[string]interface{}); ok && metaValue != nil {
						// Found metadata in the fields
						metadataField = metaValue

						// Also set it in the form.MetaData for future use
						form.MetaData = metadataField
						break
					}
				}
			}
		}

		// Check if we found valid metadata
		if metadataField == nil {
			return nil, fmt.Errorf("valid metadata is required for type=3 forms")
		}

		// Continue with validation using metadataField
		// Check required fields
		requiredFields := []string{"dataType", "taskType", "modelFramework", "modelArchitecture", "modelDatasetUrl"}
		for _, field := range requiredFields {
			if _, exists := metadataField[field]; !exists || metadataField[field] == nil {
				return nil, fmt.Errorf("missing required metadata field: %s", field)
			}
		}

		// Validate modelWeightUrl structure
		weightUrl, exists := metadataField["modelWeightUrl"]
		if !exists || weightUrl == nil {
			return nil, fmt.Errorf("modelWeightUrl is required")
		}

		weightUrlMap, ok := weightUrl.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("modelWeightUrl must be an object")
		}

		// Check if it has path OR (link and pat)
		if path, hasPath := weightUrlMap["path"]; hasPath && path != "" {
			// Valid path provided
		} else if link, hasLink := weightUrlMap["link"]; hasLink && link != "" {
			// Valid link provided, pat is optional
		} else {
			return nil, fmt.Errorf("modelWeightUrl must contain either a non-empty path or link")
		}
	}
	if form.Type == 2 {
		userID, _ := ctx.Value(middleware.CtxUserIDKey).(string)
		form.UserID = userID
	}
	orgID, _ := ctx.Value(middleware.CtxOrganizationIDKey).(string)
	if form.Type != 1 {
		if orgID != "" {
			form.OrganizationID = orgID
		}
	}
	createdForm, err := s.formRepo.CreateForm(ctx, form)
	if err != nil {
		return nil, errcom.ErrUnableToCreate
	}
	if form.Type != 1 {
		var audit dto.AuditLogs
		var datasetName string
		userID, _ := ctx.Value(middleware.CtxUserIDKey).(string)
		email, _ := ctx.Value(middleware.CtxEmailKey).(string)
		userId, err := uuid.FromString(userID)
		userDetail, err := s.userRepo.GetUserByID(ctx, userId)
		if err != nil {
			return nil, errcom.ErrUserNotFound
		}
		ctx = context.WithValue(ctx, "role", userDetail.Role.Name)
		userrole, _ := ctx.Value("role").(string)
		fmt.Println("Role", userrole)
		if createdForm.Type == 2 {
			for _, field := range createdForm.Fields {
				if field.Label == "Dataset Name" {
					if val, ok := field.Value.(string); ok {
						datasetName = val
						break
					}
				}
			}
			audit = dto.AuditLogs{
				OrganizationID: orgID,
				Timestamp:      time.Now().UTC(),
				UserID:         userID,
				UserName:       email,
				UserRole:       userrole,
				Activity:       "Created Dataset",
				Dataset:        datasetName,
				Details: map[string]string{
					"form_id":   createdForm.ID.String(),
					"form_type": fmt.Sprintf("%d", createdForm.Type),
				},
			}
			go kafkas.PublishAuditLog(&audit, os.Getenv("KAFKA_BROKER_ADDRESS"), "audit-logs")
		} else if createdForm.Type == 3 {
			var projectdocketName string
			for _, field := range createdForm.Fields {
				if field.Label == "Project Name" {
					if val, ok := field.Value.(string); ok {
						projectdocketName = val
						break
					}
					if field.Label == "Tagging to sample datasets" {
						if val, ok := field.Value.(string); ok {
							datasetName = val
							break
						}
					}
				}
			}
			audit = dto.AuditLogs{
				OrganizationID: orgID,
				Timestamp:      time.Now().UTC(),
				UserID:         userID,
				UserName:       email,
				UserRole:       "User",
				Activity:       "Created Project Docket",
				ProjectDocket:  projectdocketName,
				Dataset:        datasetName,
				Details: map[string]string{
					"form_id":   createdForm.ID.String(),
					"form_type": fmt.Sprintf("%d", createdForm.Type),
				},
			}
			go kafkas.PublishAuditLog(&audit, os.Getenv("KAFKA_BROKER_ADDRESS"), "audit-logs")
		}
	}

	// Optional: Run async
	return createdForm, err

}

func (s *service) GetFormByType(ctx context.Context, doc_type, page, limit, status int) (*model.GetFormResponse, error) {

	formList, total, err := s.formRepo.GetFormByType(ctx, doc_type, page, limit, status)
	if err != nil {
		//commonlib.LogMessage(s.logger, commonlib.Error, "GetForms", err.Error(), err, "type", doc_type)
		if errors.Is(err, errcom.ErrNotFound) {
			return &model.GetFormResponse{
				FormDtoData: make([]map[string]interface{}, 0), // empty slice, not nil
				PagingInfo: model.PagingInfo{
					TotalItems:  0,
					CurrentPage: page,
					TotalPage:   0,
					ItemPerPage: limit,
				},
			}, nil
		}
		return nil, err
	}
	if commonlib.IsEmpty(formList) {
		return &model.GetFormResponse{
			FormDtoData: make([]map[string]interface{}, 0), // empty slice, not nil
			PagingInfo: model.PagingInfo{
				TotalItems:  0,
				CurrentPage: page,
				TotalPage:   0,
				ItemPerPage: limit,
			},
		}, nil
	}
	//var result []*model.FormDTO

	// Iterate over the formList to convert the data into the desired format
	var flattenedData []map[string]interface{}

	for _, form := range formList {
		// Base fields
		entry := map[string]interface{}{
			"id":              form.ID,
			"organization_id": form.OrganizationID,
			"status":          form.Status,
			"created_at":      form.CreatedAt,
			"updated_at":      form.UpdatedAt,
			"type":            form.Type,
			"like_count":      form.Flags.LikeCount,
			"average_rating":  form.Flags.AverageRating,
		}

		// Add fields to the same map
		for _, field := range form.Fields {
			entry[field.Label] = field.Value
		}

		flattenedData = append(flattenedData, entry)
	}
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	res := &model.GetFormResponse{
		FormDtoData: flattenedData,
		PagingInfo: model.PagingInfo{
			TotalItems:  total,
			CurrentPage: page,
			TotalPage:   totalPages,
			ItemPerPage: limit,
		},
	}

	return res, nil
}

func (s *service) CreateFormType(ctx context.Context, formtype dto.FormType) (*dto.FormType, error) {
	existing, err := s.formTypeRepo.GetFormTypeByName(ctx, formtype.Name)
	if err == nil && existing != nil {
		return nil, errcom.ErrFormTypeExist
	}
	createdFormType, err := s.formTypeRepo.CreateFormType(ctx, &formtype)
	if err != nil {
		commonlib.LogMessage(s.logger, commonlib.Error, "CreateTemplate", err.Error(), err, "CreateBy", createdFormType)
		return nil, errcom.ErrUnableToCreate
	}
	return createdFormType, err
}

func (s *service) GetAllFormTypes(ctx context.Context) ([]dto.FormType, error) {
	formTypes, err := s.formTypeRepo.GetAllFormTypes(ctx)
	if err != nil {
		commonlib.LogMessage(s.logger, commonlib.Error, "GetAllFormTypes", err.Error(), err)
		return nil, errcom.ErrNotFound
	}
	if commonlib.IsEmpty(formTypes) {
		return nil, errcom.ErrRecordNotFounds
	}
	return formTypes, nil
}

func (s *service) UpdateForm(ctx context.Context, id string, status string) (*model.Response, error) {
	org, err := s.formRepo.GetFormById(ctx, id)
	if err != nil {
		return nil, errcom.ErrRecordNotFounds
	}

	orgreq := &dto.CreateOrganizationRequest{}
	for _, val := range org.Fields {
		switch val.Label {
		case "Organization Name":
			if name, ok := val.Value.(string); ok {
				orgreq.Name = name
			}
		case "Admin Email Address":
			if email, ok := val.Value.(string); ok {
				orgreq.Email = email
			}
		}
	}
	updatedForm, err := s.formRepo.UpdateForm(ctx, id, status)
	if err != nil {
		if errors.Is(err, errors.New(errcom.ErrRecordNotFound)) {
			commonlib.LogMessage(s.logger, commonlib.Error, "FormUpdate", err.Error(), nil, "form", id)
			return nil, errcom.ErrRecordNotFounds
		}
		return nil, err
	}

	// When trying to reject a pending organization
	if org.Type == 1 && org.Status == 0 && updatedForm.Status == 2 {
		return &model.Response{Message: "Form rejected"}, nil
	}

	// This is for handling already approved organization and then if we are rejecting the organization
	if org.Type == 1 && updatedForm.Status == 2 {
		for _, field := range updatedForm.Fields {
			if field.Label == "Admin Email Address" {
				fmt.Println("Found Admin Email Address field:")
				fmt.Printf("ID: %d, Placeholder: %s, Value: %v\n", field.ID, field.Placeholder, field.Value)

				// Assert that field.Value is a string
				email, ok := field.Value.(string)
				if !ok {
					return nil, errcom.ErrInvalidEmail // Or fmt.Errorf("email value is not a string")
				}
				domainParts := strings.Split(email, "@")
				if len(domainParts) != 2 {
					return nil, errcom.ErrInvalidEmail
				}
				orgDomainInForm := domainParts[1]
				orgid, err := s.organizationRepo.DeleteOrganizationByDomain(ctx, orgDomainInForm)
				if err != nil {
					return nil, errcom.ErrUnabletoDelete
				}
				errs := s.orgSettingRepo.DeleteOrganizationSettingByOrgID(ctx, orgid.String())
				if errs != nil {
					return nil, errcom.ErrUnabletoDelete
				}

			}
		}
	}
	// to get all the general setting value
	generalSettings, err := s.globalSettingRepo.GetAllGeneralSetting()
	if err != nil {
		return nil, errcom.ErrFailedToFetch
	}

	if generalSettings == nil {
		return nil, errcom.ErrRecordNotFounds
	}

	// Step 3: Use general settings
	firstSetting := generalSettings

	if status == "APPROVED" && org.Type == 1 {
		orgreq.UserCount = 0
		// based on the general seting the max count is added for organization
		orgreq.Metadata = map[string]interface{}{
			"max_user_count": firstSetting.MaxUsersPerOrganization,
		}
		organizationId, err := s.organizationRepo.CreateOrganization(ctx, orgreq)
		if err != nil {
			return nil, errcom.ErrUnableToCreate
		}
		fmt.Println("The organization is givn eas:", organizationId)

		errd := s.formRepo.UpdateOrgID(ctx, id, organizationId.OrganizationID.String())
		if errd != nil {
			return nil, errcom.ErrUnabletoUpdate
		}

		// Convert int unit to string
		unitEnum := commonlib.HASH_TO_ENUM["MaxProjectDocketSizeUnit"][firstSetting.MaxProjectDocketSizeUnit]
		if unitEnum == "" {
			unitEnum = "UNKNOWN"
		}

		// created org settings based on general setting value
		orgSetting := &entities.OrganizationSetting{
			OrgID:                    organizationId.OrganizationID,
			DefaultDeletionDays:      firstSetting.DefaultDeletionDays,
			DefaultArchivingDays:     firstSetting.DefaultArchivingDays,
			MaxActiveProjects:        firstSetting.MaxActiveProjects,
			MaxUsersPerOrganization:  firstSetting.MaxUsersPerOrganization,
			MaxProjectDocketSize:     firstSetting.MaxProjectDocketSize,
			MaxProjectDocketSizeUnit: unitEnum,
			ScheduledEvaluationTime:  firstSetting.ScheduledEvaluationTime,
		}

		// Step 6: Save organization setting
		err = s.orgSettingRepo.CreateOrganizationSetting(ctx, orgSetting)
		if err != nil {
			return nil, errcom.ErrUnableToCreate
		}

		fmt.Println("OrganizationSetting created successfully for organization ID:", organizationId)

	}

	// If it is an organization approval or reject then send an email
	sendEmail(orgreq.Email, status)

	// Final response message
	if status == "APPROVED" && updatedForm != nil {
		return &model.Response{Message: "Form successfully approved"}, nil
	}
	return &model.Response{Message: "Form rejected"}, nil
}

func (s *service) UpdateFormStatus(ctx context.Context, id string, status string) (*model.Response, error) {
	var audit dto.AuditLogs
	userID, _ := ctx.Value(middleware.CtxUserIDKey).(string)
	email, _ := ctx.Value(middleware.CtxEmailKey).(string)
	orgID, _ := ctx.Value(middleware.CtxOrganizationIDKey).(string)

	updatedForm, err := s.formRepo.UpdateForm(ctx, id, status)
	if err != nil {
		if errors.Is(err, errors.New(errcom.ErrRecordNotFound)) {
			commonlib.LogMessage(s.logger, commonlib.Error, "FormUpdate", err.Error(), nil, "form", id)
			return nil, errcom.ErrRecordNotFounds
		}
		return nil, err
	}

	if updatedForm.Type == 3 && updatedForm.Status == 9 {
		roleNames := []string{"SuperAdmin", "Collaborator"}

		roles, err := s.roleRepo.GetRolesByNames(ctx, roleNames)
		if err != nil {
			log.Println("Error getting roles:", err)
			return nil, errcom.ErrRecordNotFounds
		}

		// Step 2: Extract role IDs
		var roleIDs []string
		for _, role := range roles {
			roleIDs = append(roleIDs, role.ID.String()) // assuming role.ID is string
		}

		// Step 3: Get users by role IDs
		users, err := s.userRepo.GetUsersByRoleIDs(ctx, roleIDs)
		if err != nil {
			log.Println("Error getting users by role IDs:", err)
			return nil, errcom.ErrRecordNotFounds
		}

		// Step 4: Log or act on users
		for _, user := range users {
			go sendEmail(user.Email, status)
		}
	}

	if updatedForm.Type == 3 {
		var projectdocketName string
		var datasetname string
		for _, field := range updatedForm.Fields {
			if field.Label == "Project Name" {
				if val, ok := field.Value.(string); ok {
					projectdocketName = val
					break
				}
			}
			if field.Label == "Dataset Name" {
				if val, ok := field.Value.(string); ok {
					datasetname = val
					break
				}
			}
		}
		audit = dto.AuditLogs{
			OrganizationID: orgID,
			Timestamp:      time.Now().UTC(),
			UserID:         userID,
			UserName:       email,
			UserRole:       "Admin",
			Activity:       "Changed Status to " + status,
			ProjectDocket:  projectdocketName,
			Dataset:        datasetname,
			Details: map[string]string{
				"form_id":   updatedForm.ID.String(),
				"form_type": fmt.Sprintf("%d", updatedForm.Type),
			},
		}
		kafkas.PublishAuditLog(&audit, os.Getenv("KAFKA_BROKER_ADDRESS"), "audit-logs") // Optional: Run async in goroutine
	}
	return &model.Response{Message: "Form status updated successfully"}, nil
}

func sendEmail(receiverEmail string, status string) error {
	from := "priyadharshini.twilight@gmail.com"
	password := "rotk reak madc kwkf"
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	auth := smtp.PlainAuth("", from, password, smtpHost)
	to := []string{receiverEmail}
	message := []byte{}

	// Properly format the message
	// Properly format the HTML message

	switch status {
	case "APPROVED":
		message = []byte("From: SingHealth <" + from + ">\r\n" +
			"To: " + receiverEmail + "\r\n" +
			"Subject: Organization Approved: Exciting News Inside!\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"\r\n" +
			"<html>" +
			"<body style='font-family: Arial, sans-serif;'>" +
			"  <div style='background-color: #f4f4f4; padding: 20px;'>" +
			"    <h2 style='color: #2e6c8b;'>🎉 Congratulations! Your Organization Got Approved 🎉</h2>" +
			"    <p>Dear <strong>" + receiverEmail + "</strong>,</p>" +
			"    <p>We are thrilled to inform you that your organization has been approved!</p>" +
			"    <p>What you need to do:</p>" +
			"    <ul>" +
			"      <li><strong>Login</strong> to your account.</li>" +
			"      <li><strong>Check</strong> your organization information.</li>" +
			"      <li><strong>Start using</strong> our platform to explore all the features available for your organization.</li>" +
			"    </ul>" +
			"    <p>We are excited to have you on board. If you have any questions, feel free to contact us anytime!</p>" +
			"    <p>Best regards,</p>" +
			"    <p><strong>SingHealth Team</strong></p>" +
			"    <p style='color: #888;'>This is an automated email, please do not reply.</p>" +
			"  </div>" +
			"</body>" +
			"</html>")
	case "REJECTED":
		message = []byte("From: SingHealth <" + from + ">\r\n" +
			"To: " + receiverEmail + "\r\n" +
			"Subject: Organization Rejected: Important Information\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"\r\n" +
			"<html>" +
			"<body style='font-family: Arial, sans-serif;'>" +
			"  <div style='background-color: #f4f4f4; padding: 20px;'>" +
			"    <h2 style='color: #e74c3c;'>❌ Unfortunately, Your Organization Was Not Approved ❌</h2>" +
			"    <p>Dear <strong>" + receiverEmail + "</strong>,</p>" +
			"    <p>We regret to inform you that your organization has not been approved for the platform at this time.</p>" +
			"    <p>Here’s why:</p>" +
			"    <ul>" +
			"      <li><strong>Review the</strong> information you submitted.</li>" +
			"      <li><strong>Ensure</strong> all required fields are correctly filled.</li>" +
			"      <li><strong>Make sure</strong> your organization meets the platform’s criteria.</li>" +
			"    </ul>" +
			"    <p>If you believe this decision was made in error, or if you have any questions or concerns, please do not hesitate to reach out to us for further clarification.</p>" +
			"    <p>We encourage you to update your submission and try again in the future.</p>" +
			"    <p>Best regards,</p>" +
			"    <p><strong>SingHealth Team</strong></p>" +
			"    <p style='color: #888;'>This is an automated email, please do not reply.</p>" +
			"  </div>" +
			"</body>" +
			"</html>")

	case "DEACTIVATED":
		message = []byte("From: SingHealth <" + from + ">\r\n" +
			"To: " + receiverEmail + "\r\n" +
			"Subject: Organization Deactivated: Important Information\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"\r\n" +
			"<html>" +
			"<body style='font-family: Arial, sans-serif;'>" +
			"  <div style='background-color: #f4f4f4; padding: 20px;'>" +
			"    <h2 style='color: #e67e22;'>⚠️ Your Organization Has Been Deactivated ⚠️</h2>" +
			"    <p>Dear <strong>" + receiverEmail + "</strong>,</p>" +
			"    <p>We would like to inform you that your organization has been <strong>deactivated</strong> from the platform.</p>" +
			"    <p>This action may have been taken due to one or more of the following reasons:</p>" +
			"    <ul>" +
			"      <li>Inactivity over an extended period.</li>" +
			"      <li>Policy violations or compliance issues.</li>" +
			"      <li>Administrative review or suspension.</li>" +
			"    </ul>" +
			"    <p>If you believe this action was taken in error, or if you have any questions or concerns, please contact our support team immediately.</p>" +
			"    <p>Thank you for your understanding.</p>" +
			"    <p>Best regards,</p>" +
			"    <p><strong>SingHealth Team</strong></p>" +
			"    <p style='color: #888;'>This is an automated message. Please do not reply directly to this email.</p>" +
			"  </div>" +
			"</body>" +
			"</html>")
	case "ADHOC_REQUEST":
		message = []byte("From: SingHealth <" + from + ">\r\n" +
			"To: " + receiverEmail + "\r\n" +
			"Subject: New Ad-hoc Request Received\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"\r\n" +
			"<html>" +
			"<body style='font-family: Arial, sans-serif; background-color: #f9f9f9; padding: 20px;'>" +
			"  <div style='max-width: 600px; margin: auto; background-color: #ffffff; padding: 20px; border-radius: 8px; box-shadow: 0 2px 5px rgba(0,0,0,0.1);'>" +
			"    <h2 style='color: #2e6c8b;'>📌 Ad-hoc Request Notification</h2>" +
			"    <p>Dear <strong>" + receiverEmail + "</strong>,</p>" +
			"    <p>You have received a new <strong>ad-hoc request</strong> that requires your attention.</p>" +
			"    <p>Please visit the site to view and resolve the request at your earliest convenience.</p>" +
			"    <p>Thank you,<br><strong>SingHealth Team</strong></p>" +
			"    <p style='color: #888888; font-size: 12px;'>This is an automated email. Please do not reply to this message.</p>" +
			"  </div>" +
			"</body>" +
			"</html>")

	}

	// Send the email
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, message)
	if err != nil {
		fmt.Println("Error sending email:", err)
		return err
	}
	fmt.Println("Organization approval mail sent successfully")
	return nil
}

func (s *service) GetFilteredForms(ctx context.Context, formType int, page int, limit int, searchParam dto.SearchParam) (*[]model.GetFormResponse, error) {

	forms, total, err := s.formRepo.GetFilteredForms(ctx, formType, page, limit, searchParam)
	if err != nil {
		// commonlib.LogMessage(s.logger, commonlib.Error, "GetFilteredForms", err.Error(), err, "FormType", formType)
		if errors.Is(err, errcom.ErrNotFound) {
			response := []model.GetFormResponse{
				{
					FormDtoData: make([]map[string]interface{}, 0),
					PagingInfo: model.PagingInfo{
						TotalItems:  0,
						CurrentPage: page,
						TotalPage:   0,
						ItemPerPage: limit,
					},
				},
			}
			return &response, nil
		}
		return nil, err
	}

	if len(forms) == 0 {
		response := []model.GetFormResponse{
			{
				FormDtoData: make([]map[string]interface{}, 0),
				PagingInfo: model.PagingInfo{
					TotalItems:  0,
					CurrentPage: page,
					TotalPage:   0,
					ItemPerPage: limit,
				},
			},
		}
		return &response, nil
	}

	var flattenedData []map[string]interface{}
	for _, form := range forms {
		entry := map[string]interface{}{
			"id":              form.ID,
			"organization_id": form.OrganizationID,
			"status":          form.Status,
			"created_at":      form.CreatedAt,
			"updated_at":      form.UpdatedAt,
			"type":            form.Type,
			"like_count":      form.Flags.LikeCount,
			"average_rating":  form.Flags.AverageRating,
		}

		for _, field := range form.Fields {
			entry[field.Label] = field.Value
		}

		flattenedData = append(flattenedData, entry)
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	response := []model.GetFormResponse{
		{
			FormDtoData: flattenedData,
			PagingInfo: model.PagingInfo{
				TotalItems:  total,
				CurrentPage: page,
				TotalPage:   totalPages,
				ItemPerPage: limit,
			},
		},
	}

	return &response, nil
}

func (s *service) SearchForms(ctx context.Context, name string, page int, limit int, searchType int) (*[]model.GetFormResponse, error) {
	// Fetch forms from the repository
	forms, total, err := s.formRepo.SearchForms(ctx, name, page, limit, searchType)
	if err != nil {
		// commonlib.LogMessage(s.logger, commonlib.Error, "SearchForms", err.Error(), err, "SearchType", searchType)
		if errors.Is(err, errcom.ErrNotFound) {
			// Return response with no forms found
			response := []model.GetFormResponse{
				{
					FormDtoData: make([]map[string]interface{}, 0),
					PagingInfo: model.PagingInfo{
						TotalItems:  0, // TotalItems will be 0
						CurrentPage: page,
						TotalPage:   0,
						ItemPerPage: limit,
					},
				},
			}
			return &response, nil
		}
		return nil, err
	}

	if len(forms) == 0 {
		// Return response with no forms found
		response := []model.GetFormResponse{
			{
				FormDtoData: make([]map[string]interface{}, 0),
				PagingInfo: model.PagingInfo{
					TotalItems:  0, // TotalItems will be 0
					CurrentPage: page,
					TotalPage:   0,
					ItemPerPage: limit,
				},
			},
		}
		return &response, nil
	}

	// Prepare flattened data
	var flattenedData []map[string]interface{}
	for _, form := range forms {
		entry := map[string]interface{}{
			"id":              form.ID,
			"organization_id": form.OrganizationID,
			"status":          form.Status,
			"created_at":      form.CreatedAt,
			"updated_at":      form.UpdatedAt,
			"type":            form.Type,
			"like_count":      form.Flags.LikeCount,
			"average_rating":  form.Flags.AverageRating,
		}

		// Flatten form fields
		for _, field := range form.Fields {
			entry[field.Label] = field.Value
		}

		// Append to flattened data
		flattenedData = append(flattenedData, entry)
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	response := []model.GetFormResponse{
		{
			FormDtoData: flattenedData,
			PagingInfo: model.PagingInfo{
				TotalItems:  total, // TotalItems is set to the total count of forms
				CurrentPage: page,
				TotalPage:   totalPages,
				ItemPerPage: limit,
			},
		},
	}

	// Return the response with the forms and pagination details
	return &response, nil
}

func (s *service) ListForms(ctx context.Context, formType int, formStatus int, page int, limit int, searchParam dto.SearchParam) (*model.GetFormResponse, error) {
	var forms []dto.FormDTO
	var total int64
	var err error

	// Decide between search or filter
	// if strings.TrimSpace(searchParam.FormName) != "" {
	// 	forms, total, err = s.formRepo.SearchForms(ctx, searchParam.FormName, page, limit, formType)
	// } else {
	// 	forms, total, err = s.formRepo.GetFilteredForms(ctx, formType, page, limit, searchParam)
	// }
	userIDStr, _ := ctx.Value(middleware.CtxUserIDKey).(string)
	userID, err := uuid.FromString(userIDStr)
	if err != nil {
		log.Println("Invalid UUID format for userID")
		return nil, errcom.ErrInvalidOrMissingJWT
	}
	userDetail, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, errcom.ErrUserNotFound
	}
	ctx = context.WithValue(ctx, "role", userDetail.Role.Name)
	userrole, _ := ctx.Value("role").(string)
	fmt.Println("Role", userrole)
	forms, total, err = s.formRepo.ListForms(ctx, formType, formStatus, page, limit, searchParam)

	if err != nil {
		if errors.Is(err, errcom.ErrNotFound) {
			// Returning empty response if not found
			emptyResponse := &model.GetFormResponse{
				FormDtoData: make([]map[string]interface{}, 0),
				PagingInfo: model.PagingInfo{
					TotalItems:  0,
					CurrentPage: page,
					TotalPage:   0,
					ItemPerPage: limit,
				},
			}
			return emptyResponse, nil
		}
		return nil, err
	}

	// Handle empty results
	if len(forms) == 0 {
		// Returning empty response if no forms are found
		emptyResponse := &model.GetFormResponse{
			FormDtoData: make([]map[string]interface{}, 0),
			PagingInfo: model.PagingInfo{
				TotalItems:  0,
				CurrentPage: page,
				TotalPage:   0,
				ItemPerPage: limit,
			},
		}
		return emptyResponse, nil
	}

	// Flatten forms
	flattenedData := make([]map[string]interface{}, 0, len(forms))
	for _, form := range forms {
		entry := map[string]interface{}{
			"id":              form.ID,
			"organization_id": form.OrganizationID,
			"status":          form.Status,
			"created_at":      form.CreatedAt,
			"updated_at":      form.UpdatedAt,
			"type":            form.Type,
			"like_count":      form.Flags.LikeCount,
			"average_rating":  form.Flags.AverageRating,
		}
		for _, field := range form.Fields {
			entry[field.Label] = field.Value
		}
		flattenedData = append(flattenedData, entry)
	}

	// Calculate total pages
	totalPages := 0
	if total > 0 && limit > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	// Return as a pointer to response
	response := &model.GetFormResponse{
		FormDtoData: flattenedData,
		PagingInfo: model.PagingInfo{
			TotalItems:  total,
			CurrentPage: page,
			TotalPage:   totalPages,
			ItemPerPage: limit,
		},
	}

	return response, nil
}

func (s *service) ShortListDocket(ctx context.Context, userId string, dto dto.ShortListDTO) (bool, error) {
	interactionCtxId := common.ValueMapper("LIKE", "OperationContext", "ENUM_TO_HASH").(int)
	existingInteraction, err := s.commEventRepo.CheckForExistingInteraction(ctx, userId, dto.InteractionId, interactionCtxId)
	if existingInteraction || err != nil {
		// return false, errcom.ErrDuplicateInteraction
		return false, errcom.ErrDuplicateInteraction
	}
	// FIXME: Check if the user has already shortlisted
	err = s.commEventRepo.CreateShortList(ctx, userId, &dto)
	if err != nil {
		commonlib.LogMessage(s.logger, commonlib.Error, "ShortListDocket", err.Error(), err, "CommEvents", userId)
		return false, errcom.ErrUnableToCreate
	}
	_, err = s.UpdateFlagField(ctx, dto.InteractionId, false, 0, true)
	if err != nil {
		return false, errcom.ErrUnabletoUpdate
	}
	return true, nil
}

func (s *service) RateDocket(ctx context.Context, userId string, dto dto.RatingDTO) (bool, error) {
	interactionCtxId := common.ValueMapper("RATING", "OperationContext", "ENUM_TO_HASH").(int)
	existingInteraction, err := s.commEventRepo.CheckForExistingInteraction(ctx, userId, dto.InteractionId, interactionCtxId)
	if existingInteraction || err != nil {
		return false, errcom.ErrDuplicateInteraction
	}
	// FIXME: Check if the user has already rated
	err = s.commEventRepo.CreateRating(ctx, userId, &dto)
	if err != nil {
		commonlib.LogMessage(s.logger, commonlib.Error, "RateDocket", err.Error(), err, "CommEvents", userId)
		return false, errcom.ErrUnableToCreate
	}
	_, err = s.UpdateFlagField(ctx, dto.InteractionId, true, dto.Rating, false)
	if err != nil {
		return false, errcom.ErrUnabletoUpdate
	}
	return true, nil
}

func (s *service) GetCommentsById(ctx context.Context, interactionId string) ([]*dto.CommentData, error) {
	res, err := s.commEventRepo.GetCommentsByProjectID(ctx, interactionId)
	if err != nil {
		commonlib.LogMessage(s.logger, commonlib.Error, "RateDocket", err.Error(), err, "CommEvents", interactionId)
		return nil, errcom.ErrRecordNotFounds
	}
	return res, nil
}

func (s *service) UpdateFlagField(ctx context.Context, id string, rating bool, ratingValue int, like bool) (bool, error) {

	fmt.Println("inside the UpdateFlagField")
	// Validation: Exactly one of rating or like must be true
	if (rating && like) || (!rating && !like) {
		return false, errcom.ErrRatingAndLikeTrue
	}

	form, err := s.formRepo.GetFormById(ctx, id)
	if err != nil {
		return false, errcom.ErrRecordNotFounds
	}

	update := bson.M{}

	// Handle Rating
	if rating {
		// Validate rating value
		if ratingValue < 1 || ratingValue > 5 {
			return false, errcom.ErrInvalidRate
		}

		// Initialize flags.rating if nil
		if form.Flags.Rating == nil {
			form.Flags.Rating = map[int]int{
				5: 0,
				4: 0,
				3: 0,
				2: 0,
				1: 0,
			}
		}

		// Increment existing rating value properly
		form.Flags.Rating[ratingValue] = form.Flags.Rating[ratingValue] + 1

		// Now recalculate average rating
		totalRatings := 0
		totalScore := 0

		for star, count := range form.Flags.Rating {
			totalRatings += count
			totalScore += star * count
		}

		if totalRatings > 0 {
			form.Flags.AverageRating = totalScore / totalRatings
		} else {
			form.Flags.AverageRating = 0
		}

		update["flags.rating"] = form.Flags.Rating
		update["flags.average_rating"] = form.Flags.AverageRating
	}

	// Handle LikeCount
	if like {
		if form.Flags.LikeCount > 0 {
			form.Flags.LikeCount = form.Flags.LikeCount + 1
		} else {
			form.Flags.LikeCount = 1
		}
		update["flags.like_count"] = form.Flags.LikeCount
	}

	res, err := s.formRepo.UpdateFormFlags(ctx, id, update)
	if err != nil {
		return false, errcom.ErrUnabletoUpdate
	}

	return res, nil
}

func (s *service) DeactivateOrganization(ctx context.Context, orgID uuid.UUID, status string) error {
	// Call repository method to deactivate organization
	org, err := s.organizationRepo.DeactivateOrganization(ctx, orgID)
	if err != nil {
		return errcom.ErrDeactivateOrganizationFailed
	}
	formList, err := s.formRepo.GetFormAll(ctx, 1)
	if err != nil {
		return errcom.ErrFailedToFetch
	}
	for _, form := range formList {
		for _, field := range form.Fields {
			if field.Label == "Admin Email Address" {
				fmt.Println("Found Admin Email Address field:")
				fmt.Printf("ID: %d, Placeholder: %s, Value: %v\n", field.ID, field.Placeholder, field.Value)

				// Assert that field.Value is a string
				email, ok := field.Value.(string)
				if !ok {
					return errcom.ErrInvalidEmail // Or fmt.Errorf("email value is not a string")
				}
				domainParts := strings.Split(email, "@")
				if len(domainParts) != 2 {
					return errcom.ErrInvalidEmail
				}

				orgDomainInForm := domainParts[1]

				if org.OrganizationDomain == orgDomainInForm {
					_, err := s.formRepo.UpdateDeactivateStatus(ctx, form.ID, status)
					if err != nil {
						return errcom.ErrUnabletoUpdate
					}
					sendEmail(org.OrganizationEmail, status)
					return nil
				}
			}
		}
	}
	return nil
}

// TestKong is a simple endpoint to check if Kong is running
func (s *service) TestKong(ctx context.Context) (*model.Response, error) {
	return &model.Response{
		Message: "forms kong api up and running",
	}, nil
}

func (s *service) SendForEvaluation(ctx context.Context, docketUUID string) (*model.Response, error) {
	// Get the form by ID
	form, err := s.formRepo.GetFormById(ctx, docketUUID)
	if err != nil {
		commonlib.LogMessage(s.logger, commonlib.Error, "SendForEvaluation", err.Error(), err, "DocketUUID", docketUUID)
		return nil, errcom.ErrRecordNotFounds
	}

	// Check if the form type is 3 (docket)
	if form.Type != 3 {
		return nil, fmt.Errorf("only docket forms (type 3) can be sent for evaluation")
	}

	// Debug: Print form details
	fmt.Printf("Form ID: %s, Type: %d\n", docketUUID, form.Type)
	fmt.Printf("Form MetaData: %+v\n", form.MetaData)
	fmt.Printf("Form has %d fields\n", len(form.Fields))

	// Create a map to hold our metadata
	metadata := make(map[string]interface{})

	// Check if form.MetaData is primitive.D (BSON document)
	if bsonDoc, ok := form.MetaData.(primitive.D); ok {
		fmt.Println("Form.MetaData is primitive.D, converting to map")
		// Convert primitive.D to map
		for _, elem := range bsonDoc {
			// Special handling for nested BSON documents
			if nestedDoc, ok := elem.Value.(primitive.D); ok {
				// Convert nested primitive.D to map
				nestedMap := make(map[string]interface{})
				for _, nestedElem := range nestedDoc {
					nestedMap[nestedElem.Key] = nestedElem.Value
				}
				metadata[elem.Key] = nestedMap
			} else {
				metadata[elem.Key] = elem.Value
			}
		}
	} else if mapData, ok := form.MetaData.(map[string]interface{}); ok && mapData != nil {
		fmt.Println("Form.MetaData is already a map")
		metadata = mapData
	} else {
		fmt.Println("Form.MetaData is not in a recognized format, looking in fields")

		// Look for metadata in fields
		for _, field := range form.Fields {
			if field.Label == "MetaData" {
				fmt.Printf("Found field with label MetaData: %+v\n", field)

				// Check if field.Value is primitive.D
				if bsonDoc, ok := field.Value.(primitive.D); ok {
					fmt.Println("Field.Value is primitive.D, converting to map")
					// Convert primitive.D to map
					for _, elem := range bsonDoc {
						// Special handling for nested BSON documents
						if nestedDoc, ok := elem.Value.(primitive.D); ok {
							// Convert nested primitive.D to map
							nestedMap := make(map[string]interface{})
							for _, nestedElem := range nestedDoc {
								nestedMap[nestedElem.Key] = nestedElem.Value
							}
							metadata[elem.Key] = nestedMap
						} else if elem.Key == "modelWeightUrl" && elem.Value != nil {
							// Handle modelWeightUrl specifically if it's not a primitive.D
							// but might be another type that needs conversion
							switch v := elem.Value.(type) {
							case []interface{}:
								// If it's an array, convert to map
								weightUrlMap := make(map[string]interface{})
								for _, item := range v {
									if kvPair, ok := item.(map[string]interface{}); ok {
										if key, hasKey := kvPair["Key"]; hasKey {
											if keyStr, ok := key.(string); ok {
												weightUrlMap[keyStr] = kvPair["Value"]
											}
										}
									}
								}
								metadata["modelWeightUrl"] = weightUrlMap
							case map[string]interface{}:
								// If it's already a map, use it directly
								metadata["modelWeightUrl"] = v
							default:
								// For any other type, store as is
								metadata[elem.Key] = elem.Value
							}
						} else {
							metadata[elem.Key] = elem.Value
						}
					}
				} else if mapData, ok := field.Value.(map[string]interface{}); ok && mapData != nil {
					// If field.Value is already a map, use it directly
					metadata = mapData
				}
			}
		}
	}

	// Additional check for modelWeightUrl to ensure it's in the correct format
	if weightUrl, exists := metadata["modelWeightUrl"]; exists {
		// Check if it's an array of key-value pairs and convert to map
		if weightUrlArray, ok := weightUrl.([]interface{}); ok {
			weightUrlMap := make(map[string]interface{})
			for _, item := range weightUrlArray {
				if kvPair, ok := item.(map[string]interface{}); ok {
					if key, hasKey := kvPair["Key"]; hasKey {
						if keyStr, ok := key.(string); ok {
							weightUrlMap[keyStr] = kvPair["Value"]
						}
					}
				}
			}
			metadata["modelWeightUrl"] = weightUrlMap
		}
	}

	// Check if we found valid metadata
	if len(metadata) == 0 {
		return nil, fmt.Errorf("docket has invalid or missing metadata")
	}

	fmt.Printf("Final metadata map: %+v\n", metadata)

	// Validate required metadata fields
	requiredFields := []string{"dataType", "taskType", "modelFramework", "modelArchitecture", "modelWeightUrl", "modelDatasetUrl"}
	for _, field := range requiredFields {
		if _, exists := metadata[field]; !exists {
			return nil, fmt.Errorf("missing required metadata field: %s", field)
		}
	}

	// // Create UUID from string
	// _, err = uuid.FromString(docketUUID)
	// if err != nil {
	// 	return nil, fmt.Errorf("invalid docket UUID format: %w", err)
	// }

	// Create entry in docket_status table with status as PENDING
	docketStatusReq := &dto.CreateDocketStatusRequest{
		DocketId: docketUUID,
		Status:   "PENDING",
	}

	CreateDocketStatus, err := s.docketStatusRepo.CreateDocketStatus(ctx, docketStatusReq)
	if err != nil {
		commonlib.LogMessage(s.logger, commonlib.Error, "SendForEvaluation", err.Error(), err, "DocketUUID", docketUUID)
		return nil, fmt.Errorf("failed to create docket status: %w", err)
	}

	// Add the docket UUID to the metadata
	metadata["uuid"] = CreateDocketStatus.ID
	fmt.Println("add dcket id in metadata")
	metadata["docket_uuid"] = docketUUID

	// Publish metadata to Kafka
	err = publishDocketMetadata(metadata)
	if err != nil {
		commonlib.LogMessage(s.logger, commonlib.Error, "SendForEvaluation", "Failed to publish to Kafka", err, "DocketUUID", docketUUID)
		return nil, fmt.Errorf("failed to publish metadata to Kafka: %w", err)
	}

	return &model.Response{Message: "Docket sent for evaluation successfully"}, nil
}

// publishDocketMetadata publishes the docket metadata to Kafka using the external Kafka service
func publishDocketMetadata(metadata map[string]interface{}) error {
	// Get Kafka broker address from environment variable
	brokerAddress := os.Getenv("KAFKA_EXT_BROKER_ADDRESS")
	if brokerAddress == "" {
		brokerAddress = "localhost:9092" // Default if not set
	}

	// Create the topic if it doesn't exist
	topic := "send-docket-for-evaluation"

	// Create a Kafka writer for the topic
	writer := kafkas.GetKafkaWriter(topic, brokerAddress)
	defer writer.Close()

	// Create a message payload
	message := map[string]interface{}{
		// "docket_uuid": docketUUID,
		"metadata":  metadata,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	// Convert the message to JSON
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message to JSON: %w", err)
	}
	// Write the message to Kafka
	var uuidKey string
	if uuidValue, ok := metadata["uuid"].(string); ok {
		uuidKey = uuidValue
	} else {
		// If uuid is not a string, convert it to string
		uuidKey = fmt.Sprintf("%v", metadata["uuid"])
	}

	err = writer.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(uuidKey),
		Value: messageJSON,
	})
	if err != nil {
		return fmt.Errorf("failed to write message to Kafka: %w", err)
	}

	return nil
}

func (s *service) GetDocketMetrics(ctx context.Context, id string) (*dto.DocketMetricsDTO, error) {
	// Get the docket status by internal UUID
	docketStatus, err := s.docketStatusRepo.GetDocketStatusByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch docket status: %w", err)
	}

	metricID := docketStatus.DocketMetricsId
	if metricID == "" {
		return nil, fmt.Errorf("docket status does not contain a valid DocketMetricsId")
	}

	metrics, err := s.docketMetricsRepo.GetByID(ctx, metricID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch docket metrics: %w", err)
	}

	return &dto.DocketMetricsDTO{
		ID:             metrics.ID,
		Metadata:       metrics.Metadata,
		CreatedAt:      metrics.CreatedAt,
		UpdatedAt:      metrics.UpdatedAt,
		DocketStatusID: metrics.DocketStatusID, // ✅ Correct field name
	}, nil
}
func (s *service) GetAllDocketDetails(ctx context.Context, search string, page, limit int) (*model.GetAllDocketDetailsResponse, error) {

	entityList, total, err := s.docketmetricsRepo.GetAllDocketDetails(ctx, search, page, limit)
	if err != nil {
		return nil, fmt.Errorf("service error: %w", err)
	}
	if len(entityList) == 0 {
		return nil, errcom.ErrRecordNotFounds
	}

	// Calculate total pages
	totalPages := 0
	if total > 0 && limit > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	// Return data and paging info
	response := &model.GetAllDocketDetailsResponse{
		Data: entityList,
		PagingInfo: model.PagingInfo{
			TotalItems:  total,
			CurrentPage: page,
			TotalPage:   totalPages,
			ItemPerPage: limit,
		},
	}

	return response, nil
}
func (s *service) AddDocketDetails(ctx context.Context, req *entities.ModelConfig) (*entities.ModelConfig, error) {
	return s.docketmetricsRepo.AddDocketDetails(ctx, req)
}

func (s *service) GetFormByID(ctx context.Context, id string) (*dto.FormDTO, error) {
	form, err := s.formRepo.GetFormById(ctx, id)
	if err != nil {
		return nil, errcom.ErrRecordNotFounds
	}

	// Return the full DTO directly
	return form, nil
}
func (s *service) UpdateFormById(ctx context.Context, form dto.FormDTO) (*dto.FormDTO, error) {
	return s.formRepo.UpdateFormById(ctx, form)
}
func (s *service) GetDocketDetailByID(ctx context.Context, id uuid.UUID) (*entities.ModelConfig, error) {
	return s.docketmetricsRepo.GetDocketDetailByID(ctx, id)
}

func (s *service) GetOrgSummaryDetail(ctx context.Context, orgID string) (*dto.OrgSummaryDetails, error) {
	result, err := s.formRepo.GetOrgSummaryDetails(ctx, orgID)
	if err != nil {
		return nil, errcom.ErrRecordNotFounds
	}
	usercount, err := s.userRepo.GetUserCountByOrganisationID(ctx, orgID)
	if err != nil {
		return nil, errcom.ErrRecordNotFounds
	}
	result.TotalUsercount = usercount

	return result, nil
}
