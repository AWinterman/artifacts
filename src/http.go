package artifacts

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
)



func initRouting(storage *RethinkStorage) *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/list", func(writer http.ResponseWriter, request *http.Request) {
		list, err := fetchArtifactsForQuery(writer, request, storage)
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

	r.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		query, err := fetchArtifactsForQuery(writer, request, storage)
		if err != nil {
			return
		}

		renderTemplate(writer, "listing", ListHtmlContext{
			Artifacts: query,
			Statuses: AllStatuses,
		})

	})

	// Add handler for static files
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("static")))

	return r
}

func NewServer(addr string, storage *RethinkStorage) *http.Server {
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
}

func fetchArtifactsForQuery(writer http.ResponseWriter, request *http.Request, storage *RethinkStorage) ([]Artifact, error) {
	rawStatus := request.URL.Query()["status"]
	namespace := request.URL.Query().Get("namespace")
	packs := request.URL.Query().Get("package")
	log.Info().Msgf("Got query %v", request.URL.Query())

	status := make([]Status, len(rawStatus))

	for i, s := range rawStatus {
		status[i] = Status(s)
	}

	list, err := storage.List(status, namespace, packs)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
	writer.WriteHeader(200)
	return list, err
}


var templates *template.Template

func init() {
	filenames := make([]string, 0)

	err := filepath.Walk("templates", func(path string, info os.FileInfo, err error) error {
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
	err := templates.ExecuteTemplate(w, tmpl + ".gohtml", vars)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

