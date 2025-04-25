package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/smtp"
	"strings"

	errcom "github.com/PecozQ/aimx-library/apperrors"
	commonlib "github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
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

		existingOrg, err := s.organizationRepo.GetOrganizationByDomain(ctx, orgDomain)
		if err != nil {
			return nil, errcom.ErrInvalidEmail
		}
		if !commonlib.IsEmpty(existingOrg) {
			fmt.Println("the existing org is given as:", existingOrg)
			return nil, errcom.ErrDuplicateEmail
		}
	}

	createdForm, err := s.formRepo.CreateForm(ctx, form)
	if err != nil {
		commonlib.LogMessage(s.logger, commonlib.Error, "CreateTemplate", err.Error(), err, "CreateBy", createdForm)
		return nil, err
	}
	return createdForm, err
}

func (s *service) GetFormByType(ctx context.Context, doc_type, page, limit int) (*model.GetFormResponse, error) {

	formList, total, err := s.formRepo.GetFormByType(ctx, doc_type, page, limit)
	if err != nil {
		//commonlib.LogMessage(s.logger, commonlib.Error, "GetForms", err.Error(), err, "type", doc_type)
		return nil, err
	}
	if commonlib.IsEmpty(formList) {
		return nil, err
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
		return nil, errors.New("Form Type Already Exists")
	}
	createdFormType, err := s.formTypeRepo.CreateFormType(ctx, &formtype)
	if err != nil {
		commonlib.LogMessage(s.logger, commonlib.Error, "CreateTemplate", err.Error(), err, "CreateBy", createdFormType)
		return nil, err
	}
	return createdFormType, err
}

func (s *service) GetAllFormTypes(ctx context.Context) ([]dto.FormType, error) {
	formTypes, err := s.formTypeRepo.GetAllFormTypes(ctx)
	if err != nil {
		commonlib.LogMessage(s.logger, commonlib.Error, "GetAllFormTypes", err.Error(), err)
		return nil, NewCustomError(errcom.ErrNotFound, err)
	}
	if commonlib.IsEmpty(formTypes) {
		return nil, NewCustomError(errcom.ErrNotFound, err)
	}
	fmt.Println("**************************", formTypes)
	return formTypes, nil
}

func (s *service) UpdateForm(ctx context.Context, id string, status string) (*model.Response, error) {

	org, err := s.formRepo.GetFormById(ctx, id)
	fmt.Println("The organization is givn eas: %v", org)
	if err != nil {
		return nil, NewCustomError(errcom.ErrNotFound, err)
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
	if status == "APPROVED" {
		orgreq.UserCount = 25
		organizationId, err := s.organizationRepo.CreateOrganization(ctx, orgreq)
		if err != nil {
			return nil, NewCustomError(errcom.ErrNotFound, err)
		}
		fmt.Println("The organization is givn eas:", organizationId)
	}
	updatedForm, err := s.formRepo.UpdateForm(ctx, id, status)
	if err != nil {
		if errors.Is(err, errors.New(errcom.ErrRecordNotFound)) {
			commonlib.LogMessage(s.logger, commonlib.Error, "FormUpdate", err.Error(), nil, "form", id)
			return nil, NewCustomError(errcom.ErrNotFound, err)
		}
		return nil, err
	}
	sendEmail(orgreq.Email, status)

	// Final response message
	if status == "APPROVED" && updatedForm {
		return &model.Response{Message: "Form successfully approved"}, nil
	}
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
func (s *service) GetFilteredFormsFilter(ctx context.Context, formType int, searchParam dto.SearchParam) ([]*dto.FormDTO, int64, error) {
	fmt.Println("*************ggggg*******", formType, searchParam)
	forms, total, err := s.formRepo.GetFilteredForms(ctx, formType, searchParam)
	if err != nil {
		commonlib.LogMessage(s.logger, commonlib.Error, "GetFilteredForms", err.Error(), err, "FormType", formType)
		return nil, 0, err
	}
	fmt.Println("&&&&&&&&&&&&&&", forms)
	return forms, total, nil
}
