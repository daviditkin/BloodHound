package handlers

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"slices"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/specterops/bloodhound/headers"
	"github.com/specterops/bloodhound/mediatypes"
	"github.com/specterops/bloodhound/src/api"
	"github.com/specterops/bloodhound/src/app"
	"github.com/specterops/bloodhound/src/ctx"
	"github.com/specterops/bloodhound/src/model"
	ingestModel "github.com/specterops/bloodhound/src/model/ingest"

	"github.com/specterops/bloodhound/src/utils"
)

// Handlers stores dependencies of all handlers (currently just the BHEApp interface)
type Handlers struct {
	bhApp app.BHApp
}

// NewHandlers initializes a Handlers struct with a BHEApp
func NewHandlers(bhApp app.BHApp) Handlers {
	return Handlers{
		bhApp: bhApp,
	}
}

const FileUploadJobIdPathParameterName = "file_upload_job_id"

// todo: handler copied from resources. push stuff into the app layer
func (s Handlers) ProcessFileUpload(response http.ResponseWriter, request *http.Request) {
	var (
		requestId             = ctx.FromRequest(request).RequestID
		fileUploadJobIdString = mux.Vars(request)[FileUploadJobIdPathParameterName]
	)

	if request.Body != nil {
		defer request.Body.Close()
	}

	// todo: handler needs to be just validation and pass it through to the app.
	if !isValidContentTypeForUpload(request.Header) {
		api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, "Content type must be application/json or application/zip", request), response)
	} else if fileUploadJobID, err := strconv.Atoi(fileUploadJobIdString); err != nil {
		api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, api.ErrorResponseDetailsIDMalformed, request), response)
	} else if fileType, validationStrategy, err := matchHeaders(request.Header); err != nil {
		api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, fmt.Sprintf("Error matching headers: %v", err), request), response)
	} else if fileUploadJob, err := s.bhApp.GetFileUploadJobByID(request.Context(), int64(fileUploadJobID)); err != nil {
		api.HandleDatabaseError(request, response, err)
	} else if fileName, err := s.bhApp.SaveIngestFile(request.Body, fileType, validationStrategy); errors.Is(err, app.ErrFileValidation) {
		api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, fmt.Sprintf("File validation failed: %v", err), request), response)
	} else if err != nil {
		api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusInternalServerError, fmt.Sprintf("Error saving ingest file: %v", err), request), response)
	} else if _, err = s.bhApp.CreateIngestTask(request.Context(), fileName, fileType, requestId, int64(fileUploadJobID)); errors.Is(err, app.ErrGenericDatabaseFailure) {
		api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusInternalServerError, api.ErrorResponseDetailsInternalServerError, request), response)
	} else if errors.Is(err, app.ErrNotFound) {
		api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusNotFound, "Ingest task not found", request), response)
	} else if err != nil {
		api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusInternalServerError, fmt.Sprintf("Error creating ingest task: %v", err), request), response)
	} else if err = s.bhApp.TouchFileUploadJobLastIngest(request.Context(), fileUploadJob); err != nil {
		api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusInternalServerError, api.ErrorResponseDetailsInternalServerError, request), response)
	} else {
		response.WriteHeader(http.StatusAccepted)
	}
}

func isValidContentTypeForUpload(header http.Header) bool {
	rawValue := header.Get(headers.ContentType.String())
	if rawValue == "" {
		return false
	} else if parsed, _, err := mime.ParseMediaType(rawValue); err != nil {
		return false
	} else {
		return slices.Contains(ingestModel.AllowedFileUploadTypes, parsed)
	}
}

func matchHeaders(header http.Header) (model.FileType, app.FileValidator, error) {
	if utils.HeaderMatches(header, headers.ContentType.String(), mediatypes.ApplicationJson.String()) {
		return model.FileTypeJson, app.WriteAndValidateJSON, nil
	} else if utils.HeaderMatches(header, headers.ContentType.String(), ingestModel.AllowedZipFileUploadTypes...) {
		return model.FileTypeZip, app.WriteAndValidateZip, nil
	} else {
		//We should never get here since this is checked a level above
		return model.FileTypeJson, nil, fmt.Errorf("invalid content type for ingest file")
	}
}
