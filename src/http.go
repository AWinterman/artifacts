package artifacts

import (
	json "encoding/json"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func initRouting(storage Storage) *mux.Router {

	r := mux.NewRouter()

	r.Methods("GET").Headers("Content-Type", "application/json").Path("/").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		list, _, _, err := fetchArtifactsForQuery(request, storage)
		jsonObjects, err := json.Marshal(list)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
		}

		bytesWritten, err := writer.Write(jsonObjects)
		log.Printf("Sent %d bytes", bytesWritten)

		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
		}
	})

	r.Methods("PUT").Path("/").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body := request.Body
		length := request.ContentLength
		bytes := make([]byte, length)
		_, err := body.Read(bytes)
		if err != io.EOF {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			log.Error().Err(err).Msgf("Failed to read bytes")

			return
		}
		artifacts := &[]Artifact{}
		err = json.Unmarshal(bytes, artifacts)
		if err != io.EOF && err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			log.Error().Err(err).Msgf("Request failed to unmarshal")
			return
		}

		insert, err := storage.Insert(*artifacts...)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			log.Error().Err(err).Msgf("Insert failed")
			return
		}
		for _, artifact := range insert {
			artifact.populateProblems()
		}
		marshal, err := json.Marshal(insert)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			log.Error().Err(err).Msgf("Response failed to marshal")
			return
		}

		_, err = writer.Write(marshal)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	r.Methods("GET").Path("/").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		listing, ns, ps, err := fetchArtifactsForQuery(request, storage)
		if err != nil {
			http.Error(writer, "Failed to load artifacts for query", 500)
		}

		renderTemplate(writer, "listing", ListHtmlContext{
			Artifacts: listing,
			Statuses:  AllStatuses,
			Namespace: ns,
			Package:   ps,
		})

	})

	// Add handler for static files
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("static")))

	return r
}

func NewServer(addr string, storage Storage) *http.Server {
	// Setup router
	router := initRouting(storage)

	// Create and start server
	return &http.Server{
		Addr:    addr,
		Handler: router,
	}
}

func StartServer(server *http.Server) {
	log.Info().Msgf("Starting server %s", server.Addr)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal().Msgf("Error: %v", err)
	}
}

type ListHtmlContext struct {
	Artifacts []Artifact
	Statuses  []Status
	Namespace string
	Package   string
}

func fetchArtifactsForQuery(request *http.Request, storage Storage) ([]Artifact, string, string, error) {
	rawStatus := request.URL.Query()["status"]
	namespace := request.URL.Query().Get("namespace")
	packs := request.URL.Query().Get("package")
	log.Info().Msgf("Got query %v", request.URL.Query())
	status := make([]Status, 0)

	for _, s := range rawStatus {
		if s == "" {
			continue
		}
		status = append(status, Status(s))
	}

	if len(status) == 0 {
		status = AllStatuses
	}

	list, err := storage.List(status, namespace, packs)
	return list, namespace, packs, err
}

var templates *template.Template

func LoadTemplates(specification Specification) {
	filenames := make([]string, 0)

	err := filepath.Walk(specification.Templates, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatal().Err(err).Msgf("Failed to walk templates")
		}
		if !info.IsDir() && filepath.Ext(path) == ".gohtml" {
			filenames = append(filenames, path)
		}

		return nil
	})

	if err != nil {
		log.Fatal().Err(err)
	}

	if len(filenames) == 0 {
		return
	}

	templates, err = template.ParseFiles(filenames...)
	if err != nil {
		log.Fatal().Err(err)
	}
}

func renderTemplate(w http.ResponseWriter, tmpl string, vars interface{}) {
	err := templates.ExecuteTemplate(w, tmpl+".gohtml", vars)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
