package main

import (
	"github.com/julienschmidt/httprouter"
	"github.com/mhannig/gitbase"

	"encoding/json"
	"net/http"

	"log"
	"strconv"
	"time"
)

type ApiContext struct {
	Repository *gitbase.Repository
}

type ApiResponse struct {
	ContentType string
	Status      int
	Body        []byte
}

func NewJsonReponse(body []byte, status int) *ApiResponse {
	return &ApiResponse{
		ContentType: "application/json",
		Status:      status,
		Body:        body,
	}
}

func JsonSuccess(body interface{}) *ApiResponse {
	payload, err := json.Marshal(body)
	if err != nil {
		log.Println("Serialization Error:", err)
		payload = []byte("could not serialize json body")
	}
	return NewJsonReponse(payload, 200)
}

func JsonError(body interface{}, code int) *ApiResponse {
	payload, err := json.Marshal(body)
	if err != nil {
		log.Println("Serialization Error:", err)
		payload = []byte("could not serialize json body")
	}

	return NewJsonReponse(payload, code)
}

func RawSuccess(body []byte) *ApiResponse {
	return &ApiResponse{
		Status: 200,
		Body:   body,
	}
}

type ApiHandle func(
	context *ApiContext,
	req *http.Request,
	params httprouter.Params) *ApiResponse

func apiRegisterRoutes(
	context *ApiContext,
	router *httprouter.Router,
) *httprouter.Router {

	// Documents
	router.GET("/api/v1/:collection/:id/:key",
		apiEndpoint(context, apiArchiveGetDocument))
	router.GET("/api/v1/:collection/:id/:key/revisions",
		apiEndpoint(context, apiArchiveGetDocumentRevisions))
	router.POST("/api/v1/:collection/:id/:key",
		apiEndpoint(context, apiArchiveUpdateDocument))
	router.PUT("/api/v1/:collection/:id/:key",
		apiEndpoint(context, apiArchiveUpdateDocument))
	router.DELETE("/api/v1/:collection/:id/:key",
		apiEndpoint(context, apiArchiveDeleteDocument))

	// Archives :: Documents
	router.GET("/api/v1/:collection/:id",
		apiEndpoint(context, apiArchiveListDocuments))
	router.DELETE("/api/v1/:collection/:id",
		apiEndpoint(context, apiArchiveDelete))

	// Archives
	router.GET("/api/v1/:collection",
		apiEndpoint(context, apiArchiveList))
	router.POST("/api/v1/:collection/:id",
		apiEndpoint(context, apiArchiveCreate))

	return router
}

//
// API Helper
//

func apiEndpoint(ctx *ApiContext, handle ApiHandle) httprouter.Handle {
	return func(
		res http.ResponseWriter,
		req *http.Request,
		params httprouter.Params) {

		// Log Request
		log.Println(req.Method, req.URL)

		// Handle request
		response := handle(ctx, req, params)

		// Set Headers based on response
		if response.ContentType != "" {
			res.Header().Add("Content-Type", response.ContentType)
		}

		res.WriteHeader(response.Status)
		res.Write(response.Body)
	}
}

//
// API Implementation
//

// API :: Archives

/*
 Get archives in collection
*/
type Archive struct {
	Id        uint64   `json:"id"`
	Documents []string `json:"documents"`
}

func apiArchiveList(
	ctx *ApiContext, req *http.Request, params httprouter.Params,
) *ApiResponse {
	collectionId := params.ByName("collection")
	collection, err := ctx.Repository.Use(collectionId)
	if err != nil {
		return JsonError(err, 500)
	}

	archives, err := collection.Archives()
	if err != nil {
		return JsonError(err, 500)
	}

	result := []Archive{}
	for _, archive := range archives {
		documents, err := archive.Documents()
		if err != nil {
			log.Println(err)
			continue
		}
		result = append(result, Archive{
			Id:        archive.Id,
			Documents: documents,
		})
	}

	return JsonSuccess(result)
}

/*
 Create archive in collection
*/
func apiArchiveCreate(
	ctx *ApiContext, req *http.Request, params httprouter.Params,
) *ApiResponse {
	collectionId := params.ByName("collection")
	collection, err := ctx.Repository.Use(collectionId)
	if err != nil {
		return JsonError(err, 500)
	}

	archive, err := collection.NextArchive("API created archive")
	if err != nil {
		return JsonError(err, 500)
	}

	_ = archive
	// Store body

	return JsonSuccess("OK")
}

/*
 Delete archive from collection
*/
func apiArchiveDelete(
	ctx *ApiContext, req *http.Request, params httprouter.Params,
) *ApiResponse {

	return JsonSuccess("OK")
}

/*
 List documents in archive
*/
func apiArchiveListDocuments(
	ctx *ApiContext, req *http.Request, params httprouter.Params,
) *ApiResponse {
	collectionId := params.ByName("collection")
	collection, err := ctx.Repository.Use(collectionId)
	if err != nil {
		return JsonError(err, 500)
	}
	archiveId, err := strconv.ParseUint(params.ByName("id"), 10, 64)

	archive, err := collection.Find(uint64(archiveId))
	if err != nil {
		return JsonError(err, 500)
	}

	documents, err := archive.Documents()
	if err != nil {
		return JsonError(err, 500)
	}

	return JsonSuccess(documents)
}

/*
 Get document from archive
*/
func apiArchiveGetDocument(
	ctx *ApiContext, req *http.Request, params httprouter.Params,
) *ApiResponse {
	collectionId := params.ByName("collection")
	collection, err := ctx.Repository.Use(collectionId)
	if err != nil {
		return JsonError(err, 500)
	}
	archiveId, err := strconv.ParseUint(params.ByName("id"), 10, 64)

	archive, err := collection.Find(uint64(archiveId))
	if err != nil {
		return JsonError(err, 500)
	}

	key := params.ByName("key")
	if key == "" {
		return JsonError("Missing parameter: key", 500)
	}

	var document []byte

	options := req.URL.Query()
	revs, ok := options["rev"]

	if !ok {
		// Fetch head
		document, err = archive.Fetch(key)
		if err != nil {
			return JsonError(err, 500)
		}
	} else {
		// Fetch with revision
		document, err = archive.FetchRevision(key, revs[0])
		if err != nil {
			return JsonError(err, 500)
		}

	}

	return RawSuccess(document)
}

/*
 Get document revisions / history
*/
type ArchiveRevision struct {
	Id        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

func apiArchiveGetDocumentRevisions(
	ctx *ApiContext, req *http.Request, params httprouter.Params,
) *ApiResponse {
	collectionId := params.ByName("collection")
	collection, err := ctx.Repository.Use(collectionId)
	if err != nil {
		return JsonError(err, 404)
	}
	archiveId, err := strconv.ParseUint(params.ByName("id"), 10, 64)

	archive, err := collection.Find(uint64(archiveId))
	if err != nil {
		return JsonError(err, 404)
	}

	key := params.ByName("key")
	if key == "" {
		return JsonError("Missing parameter: key", 500)
	}

	history, err := archive.History(key)
	if err != nil {
		return JsonError(err, 500)
	}

	result := []ArchiveRevision{}
	for _, commit := range history {
		result = append(result, ArchiveRevision{
			Id:        commit.Id,
			CreatedAt: commit.CreatedAt,
		})
	}

	return JsonSuccess(result)
}

/*
 Add / update document in / to archive
*/
func apiArchiveUpdateDocument(
	ctx *ApiContext, req *http.Request, params httprouter.Params,
) *ApiResponse {

	return JsonSuccess("OK")
}

/*
 Remove document from archive
*/
func apiArchiveDeleteDocument(
	ctx *ApiContext, req *http.Request, params httprouter.Params,
) *ApiResponse {
	return JsonSuccess("OK")
}
