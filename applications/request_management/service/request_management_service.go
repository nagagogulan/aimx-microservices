package service

import (
    "context"
	"fmt"
    "strconv"

    "github.com/PecozQ/aimx-library/common"
    "github.com/PecozQ/aimx-library/domain/dto"
    "github.com/PecozQ/aimx-library/domain/entities"
    "github.com/PecozQ/aimx-library/domain/repository"
    "github.com/gofrs/uuid"
)

type RequestService interface {
    CreateRequest(ctx context.Context, req *dto.CreateRequestDTO) error
    UpdateRequestStatus(ctx context.Context, id uuid.UUID, statusDTO *dto.UpdateRequestStatusDTO) error
    GetRequestsByOrg(ctx context.Context, orgID uuid.UUID) ([]*dto.RequestResponseDTO, error)
    GetAllRequests(ctx context.Context) ([]*dto.RequestResponseDTO, error)
    GetRequestByID(ctx context.Context, id uuid.UUID) (*dto.RequestResponseDTO, error)

}

type requestService struct {
    requestRepo repository.RequestRepository
    orgSettingRepo  repository.OrganizationSettingRepository

}

func NewRequestService(requestRepo repository.RequestRepository, 	orgSettingRepo  repository.OrganizationSettingRepository,
    ) RequestService {
    return &requestService{requestRepo: requestRepo, orgSettingRepo: orgSettingRepo,
    }
}

func (s *requestService) CreateRequest(ctx context.Context, req *dto.CreateRequestDTO) error {
    rent := &entities.RequestManagement{
        OrgID:         req.OrgID,
		Value: req.Value,
        CreatedByUserID:        req.CreatedByUserID,
        RequestType:            req.RequestType,
        Status:                 common.ENUM_TO_HASH["RequestStatus"]["PENDING"],
        CommandsByOrganization: req.Commands,
    }
	fmt.Println(rent)
    return s.requestRepo.CreateRequest(rent)
}

// func (s *requestService) UpdateRequestStatus(ctx context.Context, id uuid.UUID, statusDTO *dto.UpdateRequestStatusDTO) error {
//     return s.requestRepo.UpdateStatus(id, statusDTO.Status, statusDTO.CommentsBySingAdmin)
// }


func (s *requestService) UpdateRequestStatus(ctx context.Context, id uuid.UUID, statusDTO *dto.UpdateRequestStatusDTO) error {
    fmt.Println("called in service")

    req, err := s.requestRepo.GetByID(id)
    if err != nil || req == nil {
        return fmt.Errorf("request not found")
    }

    // Update status and comments
    err = s.requestRepo.UpdateStatus(id, statusDTO.Status, statusDTO.CommentsBySingAdmin)
    if err != nil {
        return err
    }

    if statusDTO.Status == common.ENUM_TO_HASH["RequestStatus"]["REJECTED"] {
        return nil
    }

    // Process approval logic
    if statusDTO.Status == common.ENUM_TO_HASH["RequestStatus"]["APPROVED"] {
        setting, err := s.orgSettingRepo.GetOrganizationSettingByOrgID(ctx, req.OrgID.String())
        if err != nil {
            return fmt.Errorf("organization setting not found: %w", err)
        }

        switch req.RequestType {
        case common.ENUM_TO_HASH["RequestType"]["Increase in Storege Size"]:
            setting.MaxProjectDocketSize += int64(req.Value)
        case common.ENUM_TO_HASH["RequestType"]["Increase in User Count"]:
            setting.MaxUsersPerOrganization += req.Value
        case common.ENUM_TO_HASH["RequestType"]["Increase in Number of active projects"]:
            setting.MaxActiveProjects += req.Value
        case common.ENUM_TO_HASH["RequestType"]["Increase in Default delete days"]:
            setting.DefaultDeletionDays += req.Value
        case common.ENUM_TO_HASH["RequestType"]["Increase in Default Archiving days"]:
            setting.DefaultArchivingDays += req.Value
        case common.ENUM_TO_HASH["RequestType"]["Change in Scheduled TIme"]:
            setting.ScheduledEvaluationTime = strconv.Itoa(req.Value) // or a proper formatted string like "04:00 PM"
        default:
            return fmt.Errorf("unsupported request type")
        }

        return s.orgSettingRepo.UpdateOrganizationSettingByOrgID(ctx, setting)
    }

    return nil
}

func (s *requestService) GetRequestsByOrg(ctx context.Context, orgID uuid.UUID) ([]*dto.RequestResponseDTO, error) {
    entities, err := s.requestRepo.GetRequestsByOrg(orgID)
    if err != nil {
        return nil, err
    }
    return mapToResponseDTOs(entities), nil
}

func (s *requestService) GetAllRequests(ctx context.Context) ([]*dto.RequestResponseDTO, error) {
    entities, err := s.requestRepo.GetAllRequests()
    if err != nil {
        return nil, err
    }
    return mapToResponseDTOs(entities), nil
}

func (s *requestService) GetRequestByID(ctx context.Context, id uuid.UUID) (*dto.RequestResponseDTO, error) {
	entity, err := s.requestRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, fmt.Errorf("request with id %s not found", id)
	}

	typeLabel := common.HASH_TO_ENUM["RequestType"][entity.RequestType]
	statusLabel := common.HASH_TO_ENUM["RequestStatus"][entity.Status]
	orgName := ""
	if entity.Organization != nil {
		orgName = entity.Organization.OrganizationName
	}

	return &dto.RequestResponseDTO{
		ID:                  entity.ID,
		OrganizationName:    orgName,
		RequestTypeLabel:    typeLabel,
		Value:               entity.Value,
		StatusLabel:         statusLabel,
		CommentsBySingAdmin: entity.CommentsBySingAdmin,
		Commands:            entity.CommandsByOrganization,
		CreatedAt:           entity.CreatedAt,
	}, nil
}


func mapToResponseDTOs(requests []entities.RequestManagement) []*dto.RequestResponseDTO {
    var list []*dto.RequestResponseDTO
    for _, r := range requests {
        typeLabel := common.HASH_TO_ENUM["RequestType"][r.RequestType]
        statusLabel := common.HASH_TO_ENUM["RequestStatus"][r.Status]
        orgName := ""
        if r.Organization != nil {
            orgName = r.Organization.OrganizationName
        }
        fmt.Println(r, common.HASH_TO_ENUM["RequestStatus"])

        list = append(list, &dto.RequestResponseDTO{
            ID:                  r.ID,
			Value: r.Value,
            OrganizationName:    orgName,
            RequestTypeLabel:    typeLabel,
            StatusLabel:         statusLabel,
            CommentsBySingAdmin: r.CommentsBySingAdmin,
            Commands:            r.CommandsByOrganization,
            CreatedAt:           r.CreatedAt,
        })
    }
    return list
}
