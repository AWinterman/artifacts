package artifacts

import (
	"github.com/aws/aws-sdk-go/service/codeartifact"
	"github.com/rs/zerolog/log"
	"github.com/sirupsen/logrus"
	db "gopkg.in/rethinkdb/rethinkdb-go.v6"
	"os"
	"time"
)

type Artifact struct {
	Repository string    `rethinkdb:"repository"`
	Namespace  string    `rethinkdb:"namespace"`
	Package    string    `rethinkdb:"package"`
	Version    string    `rethinkdb:"version""`
	Revision   string    `rethinkdb:"revision"`
	DomainName string    `rethinkdb:"domain"`
	Format     string    `rethinkdb:"format"`
	Id         string    `rethinkdb:"id"`
	Error      error     `rethinkdb:"-"`
	Status     Status    `rethinkdb:"status"`
	CreateTime time.Time `rethinkdb:"create_time"`
}

// Status represents package version status. See: https://docs.aws.amazon.com/codeartifact/latest/ug/packages-overview.html#package-version-status
type Status string

const (
	Published  Status = "Published"
	Unfinished Status = "Unfinished"
	Unlisted   Status = "Unlisted"
	Archived   Status = "Archived"
	Disposed   Status = "Disposed"
	Deleted    Status = "Deleted"
)

var AllStatuses = []Status{
	Published,
	Unfinished,
	Unfinished,
	Archived,
}

type Package struct {
	*codeartifact.RepositorySummary
	*codeartifact.PackageSummary
	Error error
}

type RethinkStorage struct {
	*db.Session
	dbName    string
	tableName string
}

func InitDbConnection(s Specification) (*RethinkStorage, error) {
	db.Log.Out = os.Stderr
	db.Log.Level = logrus.DebugLevel

	session, err := db.Connect(db.ConnectOpts{
		Address:  s.Rethink, // endpoint without http
		Database: "artifact",
	})
	if err != nil {
		return nil, err
	}

	storage := RethinkStorage{
		session,
		s.DbName,
		s.TableName,
	}
	err = storage.ensureTableExists()
	return &storage, err
}

func (rs *RethinkStorage) ensureTableExists() error {
	tablesCursor, err := db.DB("artifacts").TableList().Contains("artifacts").Run(rs)
	if err != nil {
		panic(err)
	}

	hasTable := false
	err = tablesCursor.One(&hasTable)
	if err != nil {
		panic(err)
	}

	if !hasTable {
		result, err := db.DB("artifacts").TableCreate("artifacts").RunWrite(rs)

		if err != nil {
			log.Fatal().Err(err)
		}
		log.Printf("Created table: %v", result)
	}
	return err
}

func (rs *RethinkStorage) Insert(artifacts ...Artifact) []error {

	errors := make([]error, len(artifacts))
	for index, artifact := range artifacts {

		term := rs.table().Insert(
			artifact, db.InsertOpts{
				Conflict: func(id, oldDoc, newDoc db.Term) interface{} {
					return newDoc
				},
			},
		)
		write, err := term.RunWrite(rs)
		errors[index] = err
		log.Printf("Wrote %+v", write)
	}
	return errors
}

func (rs *RethinkStorage) table() db.Term {
	return db.DB(rs.dbName).Table(rs.tableName)
}

func (rs *RethinkStorage) List(status []Status, namespaceSubstring, packageIdSubstring string) ([]Artifact, error) {
	results := make([]Artifact, 0)
	listQuery := rs.artifactsQuery(status, namespaceSubstring, packageIdSubstring)

	cursor, err := listQuery.Run(rs)

	if err != nil {
		return results, err
	}

	err = cursor.All(&results)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (rs *RethinkStorage) Changes(status []Status, namespaceSubstring, packageIdSubstring string, versions chan Artifact) {
	listQuery := rs.artifactsQuery(status, namespaceSubstring, packageIdSubstring)

	cursor, err := listQuery.Changes().Run(rs)

	if err != nil {
		versions <- Artifact{Error: err}
	}

	for {
		artifact := Artifact{}
		ok := cursor.Next(&artifact)
		versions <- artifact
		if !ok {
			err := cursor.Err()
			if err != nil {
				versions <- Artifact{Error: err}
			}
			close(versions)
		}
	}
}



func (rs *RethinkStorage) artifactsQuery(status []Status, namespaceSubstring string, packageIdSubstring string) db.Term {
	if len(status) == 0 {
		status = []Status{
			Published,
		}
	}
	filter := make([]interface{}, 0)

	statusFilter := make([]interface{}, len(status))
	for i, s := range status {
		statusFilter[i] = db.Row.Field("status").Eq(s)
		listQuery := rs.table().Filter(db.Row.Field("status").Eq(s))
		log.Info().Msgf("DB :%s", listQuery.Info())
		return listQuery
	}

	filter = append(filter, db.Row.Or(statusFilter...))

	if namespaceSubstring != "" {
		filter = append(filter, db.Row.Field("namespace").Contains(namespaceSubstring))
	}

	if packageIdSubstring != "" {
		filter = append(filter, db.Row.Field("package").Contains(packageIdSubstring))
	}

	listQuery := rs.table().Filter(db.Row.And(filter...)).OrderBy(db.Asc("id"), db.Desc("create_time"))
	log.Info().Msgf("query :%s", listQuery.ToJSON())

	return listQuery
}
