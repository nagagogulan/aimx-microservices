package service

import (
	"context"
	"errors"
	"fmt"
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
	kafka "github.com/PecozQ/aimx-library/kafka"
	middleware "github.com/PecozQ/aimx-library/middleware"
	"github.com/gofrs/uuid"
	"go.mongodb.org/mongo-driver/bson"
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
	}

	createdForm, err := s.formRepo.CreateForm(ctx, form)
	if err != nil {
		return nil, errcom.ErrUnableToCreate
	}
	var audit dto.AuditLogs
	userID, _ := ctx.Value(middleware.CtxUserIDKey).(string)
	email, _ := ctx.Value(middleware.CtxEmailKey).(string)
	orgID, _ := ctx.Value(middleware.CtxOrganizationIDKey).(string)
	if createdForm.Type == 2 {
		var datasetName string
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
			UserRole:       "Collaborator",
			Activity:       "Created Dataset",
			Dataset:        datasetName,
			Details: map[string]string{
				"form_id":   createdForm.ID.String(),
				"form_type": fmt.Sprintf("%d", createdForm.Type),
				"message":   "Form created successfully",
			},
		}
	} else if createdForm.Type == 3 {
		var projectdocketName string
		for _, field := range createdForm.Fields {
			if field.Label == "Project Name" {
				if val, ok := field.Value.(string); ok {
					projectdocketName = val
					break
				}
			}
		}
		audit = dto.AuditLogs{
			OrganizationID: orgID,
			Timestamp:      time.Now().UTC(),
			UserID:         userID,
			UserName:       email,
			UserRole:       "User",
			Activity:       "Form Created",
			ProjectDocket:  projectdocketName,
			Dataset:        "Created Project Docket",
			Details: map[string]string{
				"form_id":   createdForm.ID.String(),
				"form_type": fmt.Sprintf("%d", createdForm.Type),
				"message":   "Form created successfully",
			},
		}

	}

	// Optional: Run async
	go kafka.PublishAuditLog(&audit, os.Getenv("KAFKA_BROKER_ADDRESS"), "audit-logs")
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
	var audit dto.AuditLogs
	org, err := s.formRepo.GetFormById(ctx, id)
	fmt.Println("The organization is givn eas: %v", org)
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

	userID, _ := ctx.Value(middleware.CtxUserIDKey).(string)
	email, _ := ctx.Value(middleware.CtxEmailKey).(string)
	orgID, _ := ctx.Value(middleware.CtxOrganizationIDKey).(string)
	if updatedForm.Type == 2 {
		var datasetName string
		for _, field := range updatedForm.Fields {
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
			UserRole:       "Collaborator",
			Activity:       "Updated Dataset",
			Dataset:        datasetName,
			Details:        map[string]string{},
		}
	} else if updatedForm.Type == 3 {
		var projectdocketName string
		for _, field := range updatedForm.Fields {
			if field.Label == "Project Name" {
				if val, ok := field.Value.(string); ok {
					projectdocketName = val
					break
				}
			}
		}
		audit = dto.AuditLogs{
			OrganizationID: orgID,
			Timestamp:      time.Now().UTC(),
			UserID:         userID,
			UserName:       email,
			UserRole:       "User",
			Activity:       "Form Created",
			ProjectDocket:  projectdocketName,
			Dataset:        "Updated Project Docket",
			Details:        map[string]string{},
		}

	}

	// Optional: Run async
	if org.Status == 0 {
		return &model.Response{Message: "Form rejected"}, nil
	}

	if updatedForm.Status == 2 {
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
	sendEmail(orgreq.Email, status)

	// Final response message
	if status == "APPROVED" && updatedForm != nil {
		return &model.Response{Message: "Form successfully approved"}, nil
	}
	go kafka.PublishAuditLog(&audit, os.Getenv("KAFKA_BROKER_ADDRESS"), "audit-logs")
	return &model.Response{Message: "Form rejected"}, nil
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
					err := s.formRepo.UpdateDeactivateStatus(ctx, form.ID, status)
					if err != nil {
						return errcom.ErrUnabletoUpdate
					}
					return nil
				}
			}
		}
	}
	return nil
}
