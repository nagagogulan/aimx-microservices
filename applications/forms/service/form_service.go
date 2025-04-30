package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/smtp"
	"strings"

	errcom "github.com/PecozQ/aimx-library/apperrors"
	"github.com/PecozQ/aimx-library/common"
	commonlib "github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
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
	updatedForm, err := s.formRepo.UpdateForm(ctx, id, status)
	if err != nil {
		if errors.Is(err, errors.New(errcom.ErrRecordNotFound)) {
			commonlib.LogMessage(s.logger, commonlib.Error, "FormUpdate", err.Error(), nil, "form", id)
			return nil, NewCustomError(errcom.ErrNotFound, err)
		}
		return nil, err
	}

	if status != "APPROVED" || status != "REJECTED" {
		return &model.Response{Message: "Form updated successfully"}, nil
	}
	if status == "APPROVED" && org.Type == 1 {
		orgreq.UserCount = 25
		organizationId, err := s.organizationRepo.CreateOrganization(ctx, orgreq)
		if err != nil {
			return nil, NewCustomError(errcom.ErrNotFound, err)
		}
		fmt.Println("The organization is givn eas:", organizationId)
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

func (s *service) GetFilteredForms(ctx context.Context, formType int, searchParam dto.SearchParam) ([]*dto.FormDTO, int64, error) {
	forms, total, err := s.formRepo.GetFilteredForms(ctx, formType, searchParam)
	if err != nil {
		commonlib.LogMessage(s.logger, commonlib.Error, "GetFilteredForms", err.Error(), err, "FormType", formType)
		return nil, 0, err
	}
	if len(forms) == 0 {
		fmt.Println("No forms found")
		return nil, 0, errcom.ErrNotFound
	}
	return forms, total, nil
}
func (s *service) SearchFormsByOrgName(ctx context.Context, req model.SearchFormsByOrganizationRequest) (*dto.FormDTO, error) {
	// Fetch single form from the repository
	form, err := s.formRepo.SearchFormsByOrganization(ctx, req.FormName, req.Type)
	if err != nil {
		commonlib.LogMessage(s.logger, commonlib.Error, "SearchFormsByOrgName", err.Error(), err, "Organization", req.FormName)
		return nil, err
	}

	return &form, nil // Return pointer to single form
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
		return false, err
	}
	_, err = s.UpdateFlagField(ctx, dto.InteractionId, false, 0, true)
	if err != nil {
		return false, err
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
		return false, err
	}
	_, err = s.UpdateFlagField(ctx, dto.InteractionId, true, dto.Rating, false)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *service) GetCommentsById(ctx context.Context, interactionId string) ([]*dto.CommentData, error) {
	res, err := s.commEventRepo.GetCommentsByProjectID(ctx, interactionId)
	if err != nil {
		commonlib.LogMessage(s.logger, commonlib.Error, "RateDocket", err.Error(), err, "CommEvents", interactionId)
		return nil, err
	}
	return res, nil
}

func (s *service) UpdateFlagField(ctx context.Context, id string, rating bool, ratingValue int, like bool) (bool, error) {

	fmt.Println("inside the UpdateFlagField")
	// Validation: Exactly one of rating or like must be true
	if (rating && like) || (!rating && !like) {
		return false, fmt.Errorf("exactly one of 'rating' or 'like' must be true")
	}

	form, err := s.formRepo.GetFormById(ctx, id)
	if err != nil {
		return false, err
	}

	update := bson.M{}

	// Handle Rating
	if rating {
		// Validate rating value
		if ratingValue < 1 || ratingValue > 5 {
			return false, fmt.Errorf("invalid rating value: %d (must be between 1 and 5)", ratingValue)
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
		return false, err
	}

	return res, nil
}
